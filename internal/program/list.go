package program

import (
	"io"

	"github.com/rs/zerolog/log"
)

func (p *Program) List(w io.Writer) error {
	dataSources, err := p.db.RetrieveDatasources()
	if err != nil {
		return err
	}

	if len(dataSources) == 0 {
		log.Info().Msg("No data sources found, syncing...")
		if err := p.Sync(); err != nil {
			return err
		}

		dataSources, err = p.db.RetrieveDatasources()
		if err != nil {
			return err
		}
	}

	printDataSources(dataSources, w)
	return nil
}
