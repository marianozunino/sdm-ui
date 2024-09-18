package app

import (
	"fmt"
	"syscall"

	"github.com/ncruces/zenity"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

// PasswordCommand represents the method used to prompt the user for a password.
type PasswordCommand string

// Constants representing the different password command methods.
const (
	PasswordCommandZenity PasswordCommand = "zenity" // Use Zenity GUI prompt for password.
	PasswordCommandCLI    PasswordCommand = "cli"    // Use CLI prompt for password.
)

// retrievePassword attempts to retrieve the password from the keyring.
// If the password is not found or an error occurs, it prompts the user to enter the password.
func (p *App) retrievePassword() (string, error) {
	log.Debug().Msg("Retrieving password from keyring...")

	// Attempt to retrieve the password from the keyring.
	password, err := p.keyring.GetSecret(p.account)
	if err == nil {
		return password, nil
	}

	log.Debug().Err(err).Msg("Password not found in keyring or failed to retrieve. Prompting user for password...")

	// Prompt the user for a password using the specified method.
	password, err = p.askForPassword(p.passwordCommand)
	if err != nil {
		log.Error().Err(err).Msg("Failed to retrieve password from user input")
		return "", err
	}

	// Check if the user provided an empty password.
	if password == "" {
		errMsg := "Empty password provided"
		log.Warn().Msg(errMsg)
		return "", fmt.Errorf(errMsg)
	}

	// Attempt to save the newly provided password in the keyring.
	if err := p.keyring.SetSecret(p.account, password); err != nil {
		log.Error().Err(err).Msg("Failed to save password in keyring")
		return "", err
	}

	log.Debug().Msg("Password successfully retrieved and saved in keyring")
	return password, nil
}

// askForPassword prompts the user for a password based on the specified PasswordCommand method.
// It returns the password and any error encountered during the process.
func (p *App) askForPassword(pc PasswordCommand) (string, error) {
	switch pc {
	case PasswordCommandZenity:
		log.Debug().Msg("Using Zenity to prompt for password...")
		_, pwd, err := zenity.Password(zenity.Title("Type your SDM password"))
		if err != nil {
			log.Error().Err(err).Msg("Failed to retrieve password using Zenity")
		}
		return pwd, err

	case PasswordCommandCLI:
		log.Debug().Msg("Using CLI to prompt for password...")
		fmt.Print("Enter Password: ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Error().Err(err).Msg("Failed to retrieve password using CLI")
		}
		// The terminal does not print a newline after input, so we add it here.
		fmt.Println()
		return string(bytePassword), err

	default:
		errMsg := fmt.Sprintf("Unknown password command: %s", pc)
		log.Error().Msg(errMsg)
		return "", fmt.Errorf(errMsg)
	}
}
