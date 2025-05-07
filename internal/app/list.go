package app

import (
	"io"
	"regexp"
	"slices"

	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/rs/zerolog/log"
)

func (p *App) List(w io.Writer, withHeader bool) error {
	log.Debug().Msg("Retrieving sorted data sources")
	dataSources, err := p.GetSortedDataSources()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get sorted data sources")
		return err
	}
	log.Debug().Int("count", len(dataSources)).Msg("Writing data sources to output")
	p.PrintDataSources(dataSources, w, withHeader)
	return nil
}

func (p *App) applyBlacklist(dataSources []storage.DataSource) []storage.DataSource {
	if len(p.blacklistPatterns) == 0 {
		return dataSources
	}

	log.Debug().
		Strs("patterns", p.blacklistPatterns).
		Int("source_count", len(dataSources)).
		Msg("Applying blacklist patterns")

	filteredDataSources := make([]storage.DataSource, 0, len(dataSources))
	blacklistedCount := 0

	for _, ds := range dataSources {
		blacklisted := false
		for _, regex := range p.blacklistPatterns {
			if match, err := regexp.MatchString(regex, ds.Name); match {
				if err != nil {
					log.Warn().Err(err).Str("pattern", regex).Msg("Invalid regex pattern")
				}
				blacklisted = true
				blacklistedCount++
				break
			}
		}
		if !blacklisted {
			filteredDataSources = append(filteredDataSources, ds)
		}
	}

	log.Debug().
		Int("filtered_out", blacklistedCount).
		Int("remaining", len(filteredDataSources)).
		Msg("Blacklist filtering complete")

	return filteredDataSources
}

func (p *App) GetSortedDataSources() ([]storage.DataSource, error) {
	log.Debug().Msg("Retrieving data sources from database")
	dataSources, err := p.db.RetrieveDatasources()
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve data sources")
		return nil, err
	}

	log.Debug().Int("count", len(dataSources)).Msg("Retrieved data sources from database")

	if len(dataSources) == 0 {
		log.Debug().Msg("No data sources found, syncing...")
		if err := p.Sync(); err != nil {
			log.Error().Err(err).Msg("Sync failed")
			return nil, err
		}

		log.Debug().Msg("Retrieving data sources after sync")
		dataSources, err = p.db.RetrieveDatasources()
		if err != nil {
			log.Error().Err(err).Msg("Failed to retrieve data sources after sync")
			return nil, err
		}
		log.Debug().Int("count", len(dataSources)).Msg("Retrieved data sources after sync")
	}

	log.Debug().Msg("Applying blacklist filters")
	dataSources = p.applyBlacklist(dataSources)

	log.Debug().Msg("Sorting data sources by last used time")
	slices.SortFunc(dataSources, func(a, b storage.DataSource) int {
		return int(b.LRU - a.LRU)
	})

	log.Debug().Int("final_count", len(dataSources)).Msg("Finished preparing data sources")
	return dataSources, nil
}
