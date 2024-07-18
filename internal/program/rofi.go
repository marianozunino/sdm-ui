package program

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"git.sr.ht/~jcmuller/go-rofi/dmenu"
	"git.sr.ht/~jcmuller/go-rofi/entry"
	"github.com/marianozunino/sdm-ui/internal/storage"
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

	entries := p.createEntriesFromBuffer(bytesOut)
	selectedEntry, err := p.getSelectionFromDmenu(entries)
	if err != nil {
		return err
	}

	return p.handleSelectedEntry(selectedEntry)
}

func (p *Program) createEntriesFromBuffer(buf *bytes.Buffer) []*entry.Entry {
	var entries []*entry.Entry
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		entries = append(entries, entry.New(string(line)))
	}
	return entries
}

func (p *Program) getSelectionFromDmenu(entries []*entry.Entry) (string, error) {
	d := dmenu.New(
		dmenu.WithPrompt("Select Data Source"),
		dmenu.WithEntries(entries...),
	)

	ctx := context.Background()
	s, err := d.Select(ctx)
	if err != nil {
		log.Printf("[rofi] Selection error: %v", err)
		return "", err
	}

	fmt.Printf("[rofi] Output: %s\n", s)
	return s, nil
}

func (p *Program) handleSelectedEntry(selectedEntry string) error {
	fields := strings.Fields(selectedEntry)

	if len(fields) < 2 {
		notify.Notify("SDM CLI", "Resource not found 🔐", "", "")
		return nil
	}

	selectedDS := fields[0]
	fmt.Printf("[rofi] DataSource: %s\n", selectedDS)

	if selectedDS == "" {
		notify.Notify("SDM CLI", "Resource not found 🔐", "", "")
		return nil
	}

	ds, err := p.db.GetDatasource(p.account, selectedDS)
	if err != nil {
		notify.Notify("SDM CLI", "Resource not found 🔐", "", "")
		return nil
	}

	if err := p.retryCommand(func() error {
		return p.sdmWrapper.Connect(ds.Name)
	}); err != nil {
		return err
	}

	p.notifyDataSourceConnected(ds)
	return p.executeSync()
}

func (p *Program) notifyDataSourceConnected(ds storage.DataSource) {
	title := "Data Source Connected 🔌"
	message := fmt.Sprintf("%s\n📋 <b>%s</b>", ds.Name, ds.Address)

	if strings.HasPrefix(ds.Address, "http") {
		open.Start(ds.Address)
	} else {
		if clip, err := clipper.GetClipboard(clipper.Clipboards...); err != nil {
			printDebug("[clipper] Failed to get clipboard: " + err.Error())
		} else {
			clip.WriteAll(clipper.RegClipboard, []byte(ds.Address))
		}
	}

	notify.Notify("SDM CLI", title, message, "")
}

