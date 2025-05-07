package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/ncruces/zenity"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

// Common password-related errors
var (
	ErrEmptyPassword      = errors.New("empty password provided")
	ErrPasswordRetrieval  = errors.New("failed to retrieve password")
	ErrUnknownPasswordCmd = errors.New("unknown password command")
)

// PasswordCommand represents the method used to prompt the user for a password
type PasswordCommand string

// Constants representing the different password command methods
const (
	PasswordCommandZenity PasswordCommand = "zenity" // Use Zenity GUI prompt for password
	PasswordCommandCLI    PasswordCommand = "cli"    // Use CLI prompt for password
)

// retrievePassword attempts to retrieve the password from the keyring.
// If the password is not found or an error occurs, it prompts the user to enter the password.
func (p *App) retrievePassword() (string, error) {
	if p.account == "" {
		log.Error().Msg("No account provided")
		return "", fmt.Errorf("no account provided")
	}

	log.Debug().Str("account", p.account).Msg("Retrieving password from keyring")

	// Create context with timeout for keyring operations
	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to retrieve the password from the keyring
	password, err := p.keyring.GetSecret(p.account)
	if err == nil && password != "" {
		log.Debug().Str("account", p.account).Msg("Password retrieved from keyring")
		return password, nil
	}

	// Log the error but don't return it, as we'll try asking the user
	if err != nil {
		log.Debug().
			Err(err).
			Str("account", p.account).
			Msg("Failed to retrieve password from keyring")
	} else {
		log.Debug().
			Str("account", p.account).
			Msg("Empty password found in keyring")
	}

	// Prompt the user for a password
	log.Debug().Str("method", string(p.passwordCommand)).Msg("Prompting user for password")
	password, err = p.askForPassword(p.passwordCommand)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", string(p.passwordCommand)).
			Msg("Failed to retrieve password from user")
		return "", fmt.Errorf("%w: %v", ErrPasswordRetrieval, err)
	}

	// Check for empty password
	if password == "" {
		log.Warn().Msg("User provided empty password")
		return "", ErrEmptyPassword
	}

	// Store the password in the keyring
	log.Debug().Str("account", p.account).Msg("Saving password to keyring")
	if err := p.keyring.SetSecret(p.account, password); err != nil {
		log.Warn().
			Err(err).
			Str("account", p.account).
			Msg("Failed to save password in keyring")
		// Continue anyway since we have the password
	} else {
		log.Debug().Msg("Password successfully saved in keyring")
	}

	return password, nil
}

// askForPassword prompts the user for a password based on the specified PasswordCommand method
func (p *App) askForPassword(pc PasswordCommand) (string, error) {
	switch pc {
	case PasswordCommandZenity:
		log.Debug().Msg("Using Zenity to prompt for password")
		title := fmt.Sprintf("Enter password for %s", p.account)
		_, pwd, err := zenity.Password(
			zenity.Title(title),
		)
		if err != nil {
			if strings.Contains(err.Error(), "canceled") {
				log.Debug().Msg("User canceled Zenity password prompt")
				return "", fmt.Errorf("password prompt canceled by user")
			}
			log.Error().Err(err).Msg("Failed to retrieve password using Zenity")
			return "", err
		}

		return pwd, nil

	case PasswordCommandCLI:
		log.Debug().Msg("Using CLI to prompt for password")
		fmt.Printf("Enter password for %s: ", p.account)

		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println() // Add newline after password input

		if err != nil {
			log.Error().Err(err).Msg("Failed to read password from terminal")
			return "", err
		}

		return string(bytePassword), nil

	default:
		log.Error().Str("command", string(pc)).Msg("Unknown password command")
		return "", fmt.Errorf("%w: %s", ErrUnknownPasswordCmd, pc)
	}
}
