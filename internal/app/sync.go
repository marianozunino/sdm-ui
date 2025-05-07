package app

import (
	"bytes"

	"github.com/rs/zerolog/log"
)

func (p *App) Sync() error {
	log.Debug().Msg("Syncing...")

	statusesBuffer := new(bytes.Buffer)

	if err := p.RetryCommand(func() error {
		statusesBuffer.Reset()
		return p.sdmWrapper.Status(statusesBuffer)
	}); err != nil {
		log.Debug().Msg("Failed to sync with SDM")
		return err
	}

	dataSources := parseDataSources(statusesBuffer.String())
	p.db.StoreServers(dataSources)

	return nil
}
