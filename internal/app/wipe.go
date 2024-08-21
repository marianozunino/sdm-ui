package app

import (
	"github.com/rs/zerolog/log"
)

func (p *App) WipeCache() error {
	log.Debug().Msg("Wiping cache...")

	err := p.db.Wipe()

	log.Debug().Msg("Wiped cache")

	return err
}
