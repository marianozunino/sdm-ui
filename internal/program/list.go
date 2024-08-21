package program

import (
	"io"
	"regexp"
	"strings"

	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/rs/zerolog/log"
)

func (p *Program) List(w io.Writer) error {
	dataSources, err := p.db.RetrieveDatasources()

	if err != nil {
		return err
	}

	dataSources = p.applyBlacklist(dataSources)

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

func (p *Program) applyBlacklist(dataSources []storage.DataSource) []storage.DataSource {
	var filteredDataSources []storage.DataSource

	if len(p.blacklistPatterns) == 0 {
		return dataSources
	}

	log.Debug().Msgf("Applying blacklist: %v", strings.Join(p.blacklistPatterns, ", "))

	for _, ds := range dataSources {
		blacklisted := false
		for _, regex := range p.blacklistPatterns {
			if match, _ := regexp.MatchString(regex, ds.Name); match {
				blacklisted = true
				break
			}
		}
		if !blacklisted {
			filteredDataSources = append(filteredDataSources, ds)
		}
	}
	return filteredDataSources
}
