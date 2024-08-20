package program

import (
	"github.com/rs/zerolog/log"
)

func (p *Program) WipeCache() error {
	log.Debug().Msg("Wiping cache...")

	err := p.db.Wipe()

	log.Debug().Msg("Wiped cache")

	return err
}
