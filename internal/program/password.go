package program

import (
	"fmt"

	"github.com/ncruces/zenity"
	"github.com/rs/zerolog/log"
)

func (p *Program) retrievePassword() (string, error) {
	log.Debug().Msg("Retrieving password...")
	password, err := p.keyring.GetSecret(p.account)
	if err != nil {
		password, err = p.askForPassword()

		if err != nil {
			log.Debug().Msg("Failed to retrieve password")
			return "", err
		}

		if password == "" {
			log.Debug().Msg("Empty password provided")
			return "", fmt.Errorf("Please provide a password")
		}

		if err := p.keyring.SetSecret(p.account, password); err != nil {
			log.Debug().Msg("Failed to save password")
			return "", err
		}
	}
	return password, nil
}

func (p *Program) askForPassword() (string, error) {
	log.Debug().Msg("Prompting for password...")
	_, pwd, err := zenity.Password(zenity.Title("Type your SDM password"))
	return pwd, err
}
