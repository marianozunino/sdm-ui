package app

import (
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/rs/zerolog/log"
)

// Fzf displays a fuzzy finder interface with available data sources
func (p *App) Fzf() error {
	log.Debug().Msg("Starting fuzzy finder interface")

	// Get data sources
	dataSources, err := p.GetSortedDataSources()
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve data sources")
		return err
	}

	log.Debug().Int("count", len(dataSources)).Msg("Retrieved data sources for fuzzy finder")

	// Format status function
	formatStatus := func(status string) string {
		if status == "connected" {
			return "⚡"
		}
		return "🔌"
	}

	// Display fuzzy finder
	log.Debug().Msg("Displaying fuzzy finder")
	idx, err := fuzzyfinder.FindMulti(
		dataSources,
		func(i int) string {
			return formatStatus(dataSources[i].Status) + " " + dataSources[i].Name
		},
	)
	// Handle selection error
	if err != nil {
		log.Error().Err(err).Msg("Failed to get selection from fuzzy finder")
		return err
	}

	// No selection made
	if len(idx) == 0 {
		log.Debug().Msg("No selection made in fuzzy finder")
		return nil
	}

	// Get selected data source
	selectedDS := dataSources[idx[0]]
	log.Debug().
		Str("name", selectedDS.Name).
		Str("status", selectedDS.Status).
		Msg("Data source selected")

	// Connect to selected data source
	log.Debug().Str("name", selectedDS.Name).Msg("Connecting to selected data source")
	if err := p.RetryCommand(func() error {
		// Update last used timestamp
		if err := p.db.UpdateLastUsed(selectedDS); err != nil {
			log.Warn().Err(err).Str("name", selectedDS.Name).Msg("Failed to update last used timestamp")
		}

		// Connect to data source
		return p.sdmWrapper.Connect(selectedDS.Name)
	}); err != nil {
		log.Error().Err(err).Str("name", selectedDS.Name).Msg("Failed to connect to data source")
		return err
	}

	// Notify user of successful connection
	log.Debug().Str("name", selectedDS.Name).Msg("Successfully connected to data source")
	p.notifyDataSourceConnected(selectedDS)

	// Sync data sources
	log.Debug().Msg("Syncing data sources after connection")
	if err := p.Sync(); err != nil {
		log.Warn().Err(err).Msg("Failed to sync data sources after connection")
	}

	return nil
}
