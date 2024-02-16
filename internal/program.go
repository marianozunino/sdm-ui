package internal

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/marianozunino/sdm-ui/internal/libsecret"
	"github.com/marianozunino/sdm-ui/internal/sdm"
	"github.com/marianozunino/sdm-ui/internal/storage"

	"github.com/martinlindhe/notify"
	"github.com/ncruces/zenity"
	"github.com/skratchdot/open-golang/open"
	"golang.design/x/clipboard"
)

type commandType string

const (
	commandSync commandType = "sync"
	commandList commandType = "list"
	commandRofi commandType = "rofi"
)

var (
	debugMode = flag.Bool("d", false, "enable debug mode")
	usageMsg  = `Specify a command to execute:
- list <email>: List all the data sources from CACHE
- sync <email>: Sync all the data sources
- rofi <email>: Select & Connect to a data source using rofi`
)

type Program struct {
	account  string
	password string

	db         storage.Storage
	keyring    libsecret.Keyring
	sdmWrapper sdm.SDMClient
}

func NewProgram() *Program {
	return &Program{
		db: *storage.NewStorage(),
	}
}

func parseArgs() (string, []string) {
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println(usageMsg)
		os.Exit(1)
	}

	command := flag.Args()[0]
	args := flag.Args()[1:]

	return command, args
}

func (p *Program) Run() error {
	if err := checkDependencies(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	command, args := parseArgs()

	defer p.db.Close()

	if len(args) == 0 {
		return errors.New("provide an email address: sdm-ui sync <email>")
	}

	p.account = args[0]
	password, err := p.retrievePassword(p.account)
	if err != nil {
		return err
	}

	p.password = password

	if err := p.validateAccount(); err != nil {
		return err
	}

	switch commandType(command) {
	case commandSync:
		return p.executeSync()
	case commandList:
		return p.executeList(os.Stdout)
	case commandRofi:
		if err := p.executeRofi(args); err != nil {
			return err
		}
		return p.executeSync()
	default:
		return fmt.Errorf("invalid command: '%s'", command)
	}
}

func (p *Program) executeSync() error {
	fmt.Printf("[sync] Account: %s\n", p.account)
	fmt.Println("[sync] Syncing...")
	statusesBuffer := new(bytes.Buffer)

	if err := p.retryCommand(func() error {
		statusesBuffer.Reset()
		return p.sdmWrapper.Status(statusesBuffer)
	}); err != nil {
		fmt.Println("[sync] Failed to sync with SDM")
		return err
	}

	dataSources := parseDataSources(statusesBuffer.String())
	p.db.StoreServers(p.account, dataSources)

	servers := parseServers(statusesBuffer.String())
	p.db.StoreServers(p.account, servers)

	return nil
}

func (p *Program) executeList(w io.Writer) error {
	dataSources, err := p.db.RetrieveDatasources(p.account)
	if err != nil {
		return err
	}

	printDataSources(dataSources, w)
	return nil
}

func (p *Program) executeRofi(args []string) error {
	bytesOut := new(bytes.Buffer)

	if err := p.executeList(bytesOut); err != nil {
		fmt.Println("[rofi] Failed to execute list")
		return err
	}

	cmd := exec.Command("rofi", "-dmenu", "-i", "-p", "Select Data Source")
	cmd.Stdin = bytesOut
	rofiOut, err := cmd.Output()

	if err != nil {
		fmt.Println("[rofi] Failed to execute rofi")
		return nil
	}
	rofiOut = regexp.MustCompile(`\s+`).ReplaceAll(rofiOut, []byte(" "))

	fmt.Printf("[rofi] Output: %s\n", rofiOut)

	fmt.Printf("[rofi] Selected: %s\n", rofiOut)
	dataSource := strings.Split(string(rofiOut), " ")[0]
	fmt.Printf("[rofi] DataSource: %s\n", dataSource)
	dataSourcePort := strings.Split(string(rofiOut), " ")[1]
	fmt.Printf("[rofi] DataSourcePort: %s\n", dataSourcePort)

	if err := clipboard.Init(); err != nil {
		panic(err)
	}

	if dataSource != "" {
		if err := p.retryCommand(func() error {
			return p.sdmWrapper.Connect(dataSource)
		}); err != nil {
			return err
		}

		message := fmt.Sprintf("Datasource Connected 🔌: %s", dataSource)
		if strings.HasPrefix(dataSourcePort, "https:") {
			message += fmt.Sprintf(" on port %s ⚡", dataSourcePort)
			open.Start(dataSourcePort)
		} else {
			message += fmt.Sprintf(" via tunnel on port %s ⚡", dataSourcePort)
		}

		notify.Notify("SDM CLI", message, "", "")
	}

	return nil
}

func printDataSources(dataSources []storage.DataSource, w io.Writer) {
	const format = "%v\t%v\t%v\n"
	tw := new(tabwriter.Writer).Init(w, 0, 8, 2, ' ', 0)

	for _, ds := range dataSources {
		status := "🔒"
		if ds.Status == "connected" {
			status = "✅"
		}

		fmt.Fprintf(tw, format, ds.Name, ds.Address, status)
	}
	tw.Flush()
}

func (p *Program) validateAccount() error {
	status, err := p.sdmWrapper.Ready()
	if err != nil {
		return err
	}

	if status.Account != nil && *status.Account != p.account {
		fmt.Println("[login] Logged in with different account, logging out...")
		if err := p.sdmWrapper.Logout(); err != nil {
			fmt.Println("[login] Failed to logout")
			return fmt.Errorf("failed to logout: %s", err)
		}
	}

	return nil
}

func (p *Program) retryCommand(command func() error) error {
	if err := command(); err != nil {
		fmt.Println("[login] Retrying authentication...")
		sdmErr, ok := err.(sdm.SDMError)
		if ok {
			switch sdmErr.Code {
			case sdm.Unauthorized:
				notify.Notify("SDM CLI", "Unauthorized 🔑", "Retrying authentication...", "")
				return p.retryCommand(func() error {
					return p.sdmWrapper.Login(p.account, p.password)
				})
			case sdm.InvalidCredentials:
				notify.Notify("SDM CLI", "Authentication error 🔑", "Invalid credentials. Removing credentials...", "")
				p.keyring.DeleteSecret(p.account)
				return err
			default:
				notify.Notify("SDM CLI", "Authentication error 🔑", err.Error(), "")
				return err
			}
		} else {
			notify.Notify("SDM CLI", "Unexpected error 🔑", err.Error(), "")
			return err
		}
	}
	return nil
}

func (p *Program) retrievePassword(email string) (string, error) {
	password, err := p.keyring.GetSecret(email)
	if err != nil {
		password, err = p.askForPassword()
		if err != nil {
			return "", err
		}
		if err := p.keyring.SetSecret(email, password); err != nil {
			return "", err
		}
	}
	return password, nil
}

func (p *Program) askForPassword() (string, error) {
	fmt.Println("[login] Prompting for password...")
	_, pwd, err := zenity.Password(zenity.Title("Type your SDM password"))
	return pwd, err
}
