package program

import (
	"bytes"
	"fmt"

	"github.com/rs/zerolog/log"
)

func (p *Program) Sync() error {
	log.Debug().Msg("Syncing...")
	statusesBuffer := new(bytes.Buffer)

	if err := p.retryCommand(func() error {
		statusesBuffer.Reset()
		return p.sdmWrapper.Status(statusesBuffer)
	}); err != nil {
		fmt.Println("[sync] Failed to sync with SDM")
		return err
	}

	dataSources := parseDataSources(statusesBuffer.String())
	p.db.StoreServers(dataSources)

	return nil
}
