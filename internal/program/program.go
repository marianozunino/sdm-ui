package program

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/marianozunino/sdm-ui/internal/libsecret"
	"github.com/marianozunino/sdm-ui/internal/sdm"
	"github.com/marianozunino/sdm-ui/internal/storage"

	"github.com/martinlindhe/notify"
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
		sdmWrapper: sdm.SDMClient{
			Exe: "sdm",
		},
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
	defer p.db.Close()

	mustHaveDependencies()

	command, args := parseArgs()

	if len(args) == 0 {
		return errors.New("provide an email address: sdm-ui <command> <email|account>")
	}

	p.account = args[0]

	if password, err := p.retrievePassword(); err == nil {
		p.password = password
	} else {
		notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
		return err
	}

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

func (p *Program) validateAccount() error {
	status, err := p.sdmWrapper.Ready()

	if err != nil {
		return err
	}

	if status.Account != nil && *status.Account != p.account {
		fmt.Println("[login] Logged in with different account, logging out...")
		if err := p.sdmWrapper.Logout(); err != nil {
			if err.(sdm.SDMError).Code == sdm.Unauthorized {
				// we are already logged out
				return nil
			}
			return fmt.Errorf("failed to logout: %s", err)
		}
	}

	return nil
}

func printDataSources(dataSources []storage.DataSource, w io.Writer) {
	const format = "%v\t%v\t%v\n"
	tw := new(tabwriter.Writer).Init(w, 0, 8, 1, '\t', 0)

	for _, ds := range dataSources {
		status := "🔒"
		if ds.Status == "connected" {
			status = "✅"
		}

		fmt.Fprintf(tw, format, ds.Name, elipsise(ds.Address, 20), status)
	}
	tw.Flush()
}

func elipsise(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (p *Program) retryCommand(command func() error) error {
	if err := command(); err != nil {
		sdmErr, ok := err.(sdm.SDMError)
		if ok {
			switch sdmErr.Code {
			case sdm.Unauthorized:
				notify.Notify("SDM CLI", "Authenticating... 🔐", "", "")
				return p.retryCommand(func() error {
					p.sdmWrapper.Login(p.account, p.password)
					return command()
				})

			case sdm.InvalidCredentials:
				notify.Notify("SDM CLI", "Authentication error 🔐", "Invalid credentials", "")
				p.keyring.DeleteSecret(p.account)
				return err
			case sdm.ResourceNotFound:
				notify.Notify("SDM CLI", "Resource not found 🔐", err.Error(), "")
				return err
			default:
				notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
				return err
			}
		} else {
			notify.Notify("SDM CLI", "Unexpected error❗", err.Error(), "")
			return err
		}
	}
	return nil
}
