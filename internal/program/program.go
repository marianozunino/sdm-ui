package program

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/adrg/xdg"
	"github.com/marianozunino/sdm-ui/internal/libsecret"
	"github.com/marianozunino/sdm-ui/internal/logger"
	"github.com/marianozunino/sdm-ui/internal/sdm"
	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/martinlindhe/notify"
	"github.com/rs/zerolog/log"
)

type commandType string

type Program struct {
	account string

	db         *storage.Storage
	dbPath     string
	keyring    libsecret.Keyring
	sdmWrapper sdm.SDMClient
}

type Option func(*Program)

func WithAccount(account string) Option {
	return func(p *Program) {
		log.Debug().Msgf("Using account: %s", account)
		p.account = account
	}
}

func WithVerbose(verbose bool) Option {
	logger.ConfigureLogger(verbose)
	return func(p *Program) {}
}

func WithDbPath(dbPath string) Option {
	return func(p *Program) {
		p.dbPath = dbPath
	}
}

func NewProgram(opts ...Option) *Program {
	mustHaveDependencies()

	p := &Program{
		sdmWrapper: sdm.SDMClient{Exe: "sdm"},
		dbPath:     xdg.DataHome,
	}

	for _, opt := range opts {
		opt(p)
	}

	db, err := storage.NewStorage(p.account, p.dbPath)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}

	p.db = db

	return p
}

func (p *Program) validateAccount() error {
	status, err := p.sdmWrapper.Ready()
	if err != nil {
		return err
	}

	if status.Account != nil && *status.Account != p.account {
		log.Debug().Msg("Logged in with a different account, logging out...")
		if err := p.sdmWrapper.Logout(); err != nil {
			if sdErr, ok := err.(sdm.SDMError); ok && sdErr.Code == sdm.Unauthorized {
				// Already logged out
				return nil
			}
			return fmt.Errorf("failed to logout: %w", err)
		}
	}

	return nil
}

func printDataSources(dataSources []storage.DataSource, w io.Writer) {
	const format = "%v\t%v\t%v\n"
	tw := tabwriter.NewWriter(w, 0, 8, 1, '\t', 0)

	for _, ds := range dataSources {
		status := "🔒"
		if ds.Status == "connected" {
			status = "✅"
		}

		fmt.Fprintf(tw, format, ds.Name, ellipsize(ds.Address, 20), status)
	}
	tw.Flush()
}

func ellipsize(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// func (p *Program) retryCommand(command func() error) error {
// 	if err := command(); err != nil {
// 		if sdErr, ok := err.(sdm.SDMError); ok {
// 			switch sdErr.Code {
// 			case sdm.Unauthorized:
// 				notify.Notify("SDM CLI", "Authenticating... 🔐", "", "")
// 				if password, err := p.retrievePassword(); err != nil {
// 					notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
// 					return err
// 				} else {
// 					log.Debug().Msg("Logging in...")
// 					if err := p.sdmWrapper.Login(p.account, password); err != nil {
// 						p.keyring.DeleteSecret(p.account)
// 						notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
// 						return err
// 					} else {
// 						log.Debug().Msg("Logged in")
// 					}
// 					return command()
// 				}
// 			case sdm.InvalidCredentials:
// 				notify.Notify("SDM CLI", "Authentication error 🔐", "Invalid credentials", "")
// 				p.keyring.DeleteSecret(p.account)
// 				return err
// 			case sdm.ResourceNotFound:
// 				notify.Notify("SDM CLI", "Resource not found 🔐", err.Error(), "")
// 				return err
// 			default:
// 				notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
// 				return err
// 			}
// 		}
// 		notify.Notify("SDM CLI", "Unexpected error❗", err.Error(), "")
// 		return err
// 	}
// 	return nil
// }
//

func (p *Program) retryCommand(command func() error) error {
	err := command()

	if err == nil {
		return nil
	}

	sdErr, ok := err.(sdm.SDMError)

	if !ok {
		notify.Notify("SDM CLI", "Unexpected error❗", err.Error(), "")
		return err
	}

	switch sdErr.Code {
	case sdm.Unauthorized:
		return p.handleUnauthorized(command)
	case sdm.InvalidCredentials:
		return p.handleInvalidCredentials(err)
	case sdm.ResourceNotFound:
		notify.Notify("SDM CLI", "Resource not found 🔐", err.Error(), "")
		return err
	default:
		notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
		return err
	}
}

func (p *Program) handleUnauthorized(command func() error) error {
	notify.Notify("SDM CLI", "Authenticating... 🔐", "", "")

	password, err := p.retrievePassword()

	if err != nil {
		notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
		return err
	}

	log.Debug().Msg("Logging in...")

	if err := p.sdmWrapper.Login(p.account, password); err != nil {
		p.keyring.DeleteSecret(p.account)
		notify.Notify("SDM CLI", "Authentication error 🔐", err.Error(), "")
		return err
	}

	log.Debug().Msg("Logged in")
	return command()
}

func (p *Program) handleInvalidCredentials(err error) error {
	notify.Notify("SDM CLI", "Authentication error 🔐", "Invalid credentials", "")
	p.keyring.DeleteSecret(p.account)
	return err
}
