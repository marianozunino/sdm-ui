package program

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/martinlindhe/notify"
	"github.com/skratchdot/open-golang/open"
	"github.com/zyedidia/clipper"
)

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

	if dataSource != "" {
		if err := p.retryCommand(func() error {
			return p.sdmWrapper.Connect(dataSource)
		}); err != nil {
			return err
		}

		title := "Data Source Connected 🔌"
		message := fmt.Sprintf(dataSource)
		message += fmt.Sprintf("\n📋 <b>%s</b>", dataSourcePort)

		if strings.HasPrefix(dataSourcePort, "https:") {
			open.Start(dataSourcePort)
		} else {
			if clip, err := clipper.GetClipboard(clipper.Clipboards...); err != nil {
				printDebug("[clipper] Failed to get clipboard: " + err.Error())
			} else {
				clip.WriteAll(clipper.RegClipboard, []byte(dataSourcePort))
			}
		}

		notify.Notify("SDM CLI", title, message, "")
	}

	return nil
}
