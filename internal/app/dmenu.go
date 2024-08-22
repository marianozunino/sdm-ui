package app

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"git.sr.ht/~marianozunino/go-rofi/dmenu"
	"git.sr.ht/~marianozunino/go-rofi/entry"

	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/martinlindhe/notify"
	"github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/zyedidia/clipper"
)

type DMenuCommand string

const (
	DMenuCommandRofi DMenuCommand = "rofi"
	DMenuCommandWofi              = "wofi"
	DMenuCommandNoop              = "noop"
)

func (d DMenuCommand) String() string {
	return string(d)
}

func (p *App) DMenu() error {

	bytesOut := new(bytes.Buffer)

	if err := p.List(bytesOut); err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to execute list: %s", err))
		return err
	}

	entries := p.createEntriesFromBuffer(bytesOut)
	selectedEntry, err := p.getSelectionFromDmenu(entries)
	if err != nil {
		return err
	}

	return p.handleSelectedEntry(selectedEntry)
}

func (p *App) createEntriesFromBuffer(buf *bytes.Buffer) []*entry.Entry {
	var entries []*entry.Entry
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		entries = append(entries, entry.New(string(line)))
	}
	return entries
}

func (p *App) getSelectionFromDmenu(entries []*entry.Entry) (string, error) {
	d := dmenu.New(
		dmenu.WithPrompt("Select Data Source"),
		dmenu.WithEntries(entries...),
		dmenu.WithExecPath(string(p.dmenuCommand)),
	)

	ctx := context.Background()
	s, err := d.Select(ctx)
	if err != nil {
		return "", err
	}

	log.Debug().Msg(fmt.Sprintf("Output: %s", s))
	return s, nil
}

func (p *App) handleSelectedEntry(selectedEntry string) error {
	fields := strings.Fields(selectedEntry)

	if len(fields) < 2 {
		notify.Notify("SDM CLI", "Resource not found 🔐", "", "")
		return nil
	}

	selectedDS := fields[0]
	log.Info().Msg(fmt.Sprintf("DataSource: %s", selectedDS))

	if selectedDS == "" {
		notify.Notify("SDM CLI", "Resource not found 🔐", "", "")
		return nil
	}

	ds, err := p.db.GetDatasource(selectedDS)
	if err != nil {
		notify.Notify("SDM CLI", "Resource not found 🔐", "", "")
		return nil
	}

	if err := p.retryCommand(func() error {
		p.db.UpdateLastUsed(ds)
		return p.sdmWrapper.Connect(ds.Name)
	}); err != nil {
		return err
	}

	p.notifyDataSourceConnected(ds)
	return p.Sync()
}

func (p *App) notifyDataSourceConnected(ds storage.DataSource) {
	title := "Data Source Connected 🔌"
	message := fmt.Sprintf("%s\n📋 <b>%s</b>", ds.Name, ds.Address)

	if strings.HasPrefix(ds.Address, "http") {
		open.Start(ds.Address)
	} else {
		if clip, err := clipper.GetClipboard(clipper.Clipboards...); err != nil {
			log.Debug().Msg(fmt.Sprintf("Failed to get clipboard: %s", err))
		} else {
			clip.WriteAll(clipper.RegClipboard, []byte(ds.Address))
		}
	}

	notify.Notify("SDM CLI", title, message, "")
}
