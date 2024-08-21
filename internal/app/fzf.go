package app

import (
	"bytes"
	"fmt"

	"git.sr.ht/~marianozunino/go-rofi/entry"

	"github.com/rs/zerolog/log"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

func (p *App) Fzf() error {

	dataSources, err := p.GetSortedDataSources()

	if err != nil {
		return err
	}

	formatStatus := func(status string) string {
		if status == "connected" {
			return "⚡"
		}
		return "🔌"
	}

	idx, err := fuzzyfinder.FindMulti(
		dataSources,
		func(i int) string {
			return formatStatus(dataSources[i].Status) + " " + dataSources[i].Name
		},
	)

	if err != nil {
		log.Error().Msg(fmt.Sprintf("Failed to execute list: %s", err))
		return err
	}

	selectedDS := dataSources[idx[0]]
	log.Debug().Msg(fmt.Sprintf("Chosen datasource: %s", selectedDS.Name))

	if err := p.retryCommand(func() error {
		p.db.UpdateLastUsed(selectedDS)
		return p.sdmWrapper.Connect(selectedDS.Name)
	}); err != nil {
		return err
	}

	p.notifyDataSourceConnected(selectedDS)

	return p.Sync()
}

func (p *App) createEntriesFromBufferForFZF(buf *bytes.Buffer) []*entry.Entry {
	var entries []*entry.Entry
	for _, line := range bytes.Split(buf.Bytes(), []byte("\n")) {
		entries = append(entries, entry.New(string(line)))
	}
	return entries
}
