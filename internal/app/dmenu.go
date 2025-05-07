package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.sr.ht/~marianozunino/go-rofi/dmenu"
	"git.sr.ht/~marianozunino/go-rofi/entry"
	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/martinlindhe/notify"
	"github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
	"github.com/zyedidia/clipper"
)

// ErrNoSelection indicates that no selection was made in the menu
var ErrNoSelection = errors.New("no selection made")

// DMenuCommand represents the type of menu command to use
type DMenuCommand string

// Available menu commands
const (
	DMenuCommandRofi DMenuCommand = "rofi"
	DMenuCommandWofi              = "wofi"
	DMenuCommandNoop              = "noop"
)

// String returns the string representation of the menu command
func (d DMenuCommand) String() string {
	return string(d)
}

// DMenu displays a menu of available data sources and handles selection
func (p *App) DMenu() error {
	log.Debug().Str("command", p.dmenuCommand.String()).Msg("Starting dmenu interface")

	// Get data sources
	bytesOut := new(bytes.Buffer)
	if err := p.List(bytesOut, false); err != nil {
		log.Error().Err(err).Msg("Failed to list data sources")
		return err
	}

	// Create entries for dmenu
	entries := p.createEntriesFromBuffer(bytesOut)
	log.Debug().Int("entries", len(entries)).Msg("Created entries for dmenu")

	// Get selection from dmenu
	selectedEntry, err := p.getSelectionFromDmenu(entries)
	if err != nil {
		if errors.Is(err, ErrNoSelection) {
			log.Debug().Msg("No selection made in dmenu")
			return nil
		}
		log.Error().Err(err).Msg("Failed to get selection from dmenu")
		return err
	}

	// Handle the selected entry
	log.Debug().Str("selection", selectedEntry).Msg("Handling selected entry")
	return p.handleSelectedEntry(selectedEntry)
}

// createEntriesFromBuffer converts buffer lines to entry objects
func (p *App) createEntriesFromBuffer(buf *bytes.Buffer) []*entry.Entry {
	if buf == nil {
		log.Warn().Msg("Received nil buffer in createEntriesFromBuffer")
		return nil
	}

	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	entries := make([]*entry.Entry, 0, len(lines))

	for _, line := range lines {
		if len(line) > 0 {
			entries = append(entries, entry.New(string(line)))
		}
	}

	log.Debug().Int("count", len(entries)).Msg("Created entries from buffer")
	return entries
}

// getSelectionFromDmenu displays dmenu and returns the selected entry
func (p *App) getSelectionFromDmenu(entries []*entry.Entry) (string, error) {
	if len(entries) == 0 {
		log.Warn().Msg("No entries to display in dmenu")
		return "", ErrNoSelection
	}

	// Create dmenu instance
	d := dmenu.New(
		dmenu.WithPrompt("Select Data Source"),
		dmenu.WithEntries(entries...),
		dmenu.WithExecPath(string(p.dmenuCommand)),
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Debug().
		Str("command", p.dmenuCommand.String()).
		Int("entries", len(entries)).
		Msg("Displaying dmenu")

	// Get selection
	s, err := d.Select(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			// User canceled dmenu
			log.Debug().Msg("User canceled dmenu selection")
			return "", ErrNoSelection
		}
		log.Error().Err(err).Msg("Error during dmenu selection")
		return "", err
	}

	log.Debug().Str("selection", s).Msg("Selection made in dmenu")
	return s, nil
}

// handleSelectedEntry processes the selected entry from dmenu
func (p *App) handleSelectedEntry(selectedEntry string) error {
	// Parse the selected entry
	fields := strings.Fields(selectedEntry)
	if len(fields) < 2 {
		log.Warn().
			Str("selection", selectedEntry).
			Msg("Invalid selection: not enough fields")
		notify.Notify("SDM CLI", "🔐 Resource not found", "", "")
		return nil
	}

	// Get the data source name
	selectedDS := fields[0]
	log.Debug().Str("datasource", selectedDS).Msg("Selected data source")

	if selectedDS == "" {
		log.Warn().Msg("Empty data source name")
		notify.Notify("SDM CLI", "🔐 Resource not found", "", "")
		return nil
	}

	// Get the data source from the database
	ds, err := p.db.GetDatasource(selectedDS)
	if err != nil {
		log.Error().
			Err(err).
			Str("datasource", selectedDS).
			Msg("Failed to get data source from database")
		notify.Notify("SDM CLI", "🔐 Resource not found", "", "")
		return nil
	}

	// Connect to the data source
	log.Debug().
		Str("name", ds.Name).
		Str("address", ds.Address).
		Msg("Connecting to data source")

	if err := p.RetryCommand(func() error {
		// Update last used timestamp
		if err := p.db.UpdateLastUsed(ds); err != nil {
			log.Warn().
				Err(err).
				Str("datasource", ds.Name).
				Msg("Failed to update last used timestamp")
		}

		// Connect to data source
		return p.sdmWrapper.Connect(ds.Name)
	}); err != nil {
		log.Error().
			Err(err).
			Str("datasource", ds.Name).
			Msg("Failed to connect to data source")
		return err
	}

	// Notify user of successful connection
	log.Debug().
		Str("name", ds.Name).
		Msg("Successfully connected to data source")
	p.notifyDataSourceConnected(ds)

	// Sync data sources
	log.Debug().Msg("Syncing data sources after connection")
	if err := p.Sync(); err != nil {
		log.Warn().
			Err(err).
			Msg("Failed to sync data sources after connection")
	}

	return nil
}

// notifyDataSourceConnected notifies the user of a successful connection
func (p *App) notifyDataSourceConnected(ds storage.DataSource) {
	title := "🔌 Data Source Connected"
	message := fmt.Sprintf("%s\n📋 <b>%s</b>", ds.Name, ds.Address)

	// Handle web URLs by opening browser
	if strings.HasPrefix(ds.Address, "http") {
		log.Debug().
			Str("url", ds.Address).
			Msg("Opening URL in browser")
		if err := open.Start(ds.Address); err != nil {
			log.Warn().
				Err(err).
				Str("url", ds.Address).
				Msg("Failed to open URL in browser")
		}
	} else {
		// Copy address to clipboard
		log.Debug().Msg("Copying address to clipboard")
		clip, err := clipper.GetClipboard(clipper.Clipboards...)
		if err != nil {
			log.Warn().
				Err(err).
				Msg("Failed to get clipboard")
		} else {
			if err := clip.WriteAll(clipper.RegClipboard, []byte(ds.Address)); err != nil {
				log.Warn().
					Err(err).
					Msg("Failed to write to clipboard")
			}
		}
	}

	// Show desktop notification
	notify.Notify("SDM CLI", title, message, "")
	log.Debug().
		Str("name", ds.Name).
		Str("address", ds.Address).
		Msg("Data source connected notification sent")
}
