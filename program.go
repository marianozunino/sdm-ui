package main

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

	"github.com/martinlindhe/notify"
	"github.com/skratchdot/open-golang/open"
	"golang.design/x/clipboard"
)

type commandType string

const (
	syncCommand commandType = "sync"
	listCommand commandType = "list"
	rofiCommand commandType = "rofi"
)

var (
	debug = flag.Bool("d", false, "log out all the debug information")
	usage = `Specify a command to execute:
- list: List all the data sources from CACHE
- sync <email>: Sync all the data sources
- rofi <email>: Select & Connect to a data source using rofi`
)

type program struct {
	storage
}

func newProgram() *program {
	return &program{
		*newStorage(),
	}
}

func (p *program) Run(command string, args []string) error {
	defer p.storage.Close()

	printDebug(fmt.Sprintf("Command: %s", command))
	printDebug(fmt.Sprintf("Args: %v", args))

	switch commandType(command) {
	case syncCommand:
		return p.executeSync(args)
	case listCommand:
		return p.executeList(os.Stdout)
	case rofiCommand:
		if err := p.executeRofi(args); err != nil {
			return err
		}
		return p.executeSync(args)
	default:
		return fmt.Errorf("invalid command: '%s'", command)
	}
}

func (p *program) executeSync(args []string) error {
	if len(args) == 0 {
		return errors.New("Provide an email address: sdm-ui sync <email>")
	}

	email := args[0]

	printDebug(fmt.Sprintf("[sync] Account: %s", email))

	printDebug("[sync] Syncing...")
	statusesWriter := new(bytes.Buffer)

	if err := syncStatuses(statusesWriter); err != nil {
		printDebug("[sync] Failed to sync with SDM, authenticating...")

		printDebug("[sync] Fetching password from keyring...")
		password, err := readOrAskForPassword(email)

		if err != nil {
			printDebug("[sync] Failed to fetch password from keyring")
			return err
		}

		printDebug("[sync] Authenticating...")
		if err := authenticate(email, password); err != nil {
			printDebug("[sync] Failed to authenticate")
			return err
		}

		statusesWriter.Reset()
		if err := syncStatuses(statusesWriter); err != nil {
			printDebug("[sync] Failed to sync with SDM")
			return err
		}
	}

	ds := parseDataSources(statusesWriter.String())
	p.storage.storeServers(ds)

	ds = parseServers(statusesWriter.String())
	p.storage.storeServers(ds)

	return nil
}

func (p *program) executeList(w io.Writer) error {
	ds, err := p.retrieveDatasources()

	if err != nil {
		return err
	}

	printDebug(fmt.Sprintf("[list] Sending output to %v", w))
	printServers(ds, w)
	return nil
}

func (p *program) executeRofi(args []string) error {
	if len(args) == 0 {
		return errors.New("Provide an email address: sdm-ui rofi <email>")
	}

	email := args[0]
	if _, err := readOrAskForPassword(email); err != nil {
		printDebug(err.Error())
		return err
	}

	bytesOut := new(bytes.Buffer)

	if err := p.executeList(bytesOut); err != nil {
		printDebug("[rofi] Failed to execute list")
		return err
	}

	cmd := exec.Command("rofi", "-dmenu", "-i", "-p", "Select Data Source")
	cmd.Stdin = bytesOut
	rofiOut, err := cmd.Output()
	if err != nil {
		printDebug("[rofi] Failed to execute rofi")
		return nil
	}
	space := regexp.MustCompile(`\s+`)
	rofiOut = space.ReplaceAll(rofiOut, []byte(" "))

	printDebug(fmt.Sprintf("[rofi] Output: %s", rofiOut))

	printDebug(fmt.Sprintf("[rofi] Selected: %s", rofiOut))
	dataSource := strings.Split(string(rofiOut), " ")[0]
	printDebug(fmt.Sprintf("[rofi] DataSource: %s", dataSource))
	dataSourcePort := strings.Split(string(rofiOut), " ")[1]
	printDebug(fmt.Sprintf("[rofi] DataSourcePort: %s", dataSourcePort))

	if err := clipboard.Init(); err != nil {
		panic(err)
	}

	if dataSource != "" {
		if err := connectToDataSource(dataSource); err != nil {
			return err
		}

		if strings.HasPrefix(dataSourcePort, "https:") {
			notify.Notify("SDM CLI", "Datasource Connected 🔌", fmt.Sprintf("Opening %s on port %s ⚡", dataSource, dataSourcePort), "")
			open.Start(dataSourcePort)
		} else {
			notify.Notify("SDM CLI", "Datasource Connected 🔌", fmt.Sprintf("Tunnel %s on port %s ⚡", dataSource, dataSourcePort), "")
		}
	}

	return nil
}

func printServers(servers []DataSource, w io.Writer) {
	const format = "%v\t%v\t%v\n"
	tw := new(tabwriter.Writer).Init(w, 0, 8, 2, ' ', 0)

	for _, s := range servers {
		status := ""
		if s.Status == "connected" {
			status = "✅"
		} else {
			status = "🔒"
		}

		fmt.Fprintf(tw, format, s.Name, s.Address, status)
	}
	tw.Flush()
}

func printDebug(msg string) {
	if *debug {
		fmt.Printf("[DEBUG]: %s\n", msg)
	}
}

func getResult(command string) (string, error) {
	var cmd *exec.Cmd
	cmd = exec.Command("sh", "-c", command)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	result := strings.TrimRight(string(out), "\n")
	return result, err
}
