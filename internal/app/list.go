package app

import (
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/rs/zerolog/log"
)

func (p *App) List(w io.Writer) error {
	dataSources, err := p.GetSortedDataSources()
	if err != nil {
		return err
	}
	printDataSources(dataSources, w)
	return nil
}

func (p *App) applyBlacklist(dataSources []storage.DataSource) []storage.DataSource {
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

func (p *App) GetSortedDataSources() ([]storage.DataSource, error) {
	dataSources, err := p.db.RetrieveDatasources()
	if err != nil {
		return nil, err
	}

	if len(dataSources) == 0 {
		log.Info().Msg("No data sources found, syncing...")
		if err := p.Sync(); err != nil {
			return nil, err
		}

		dataSources, err = p.db.RetrieveDatasources()
		if err != nil {
			return nil, err
		}
	}

	dataSources = p.applyBlacklist(dataSources)

	// sort by LRU
	slices.SortFunc(dataSources, func(a, b storage.DataSource) int {
		return int(b.LRU - a.LRU)
	})

	return dataSources, nil
}
