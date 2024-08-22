package app

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

type App struct {
	account string

	db                *storage.Storage
	dbPath            string
	keyring           libsecret.Keyring
	sdmWrapper        sdm.SDMClient
	dmenuCommand      DMenuCommand
	blacklistPatterns []string
}

type AppOption func(*App)

func WithAccount(account string) AppOption {
	return func(p *App) {
		log.Debug().Msgf("Using account: %s", account)
		p.account = account
	}
}

func WithVerbose(verbose bool) AppOption {
	logger.ConfigureLogger(verbose)
	return func(p *App) {}
}

func WithDbPath(dbPath string) AppOption {
	return func(p *App) {
		p.dbPath = dbPath
	}
}

func WithBlacklist(patterns []string) AppOption {
	return func(p *App) {
		p.blacklistPatterns = patterns
	}
}

func WithCommand(command DMenuCommand) AppOption {
	return func(p *App) {
		p.dmenuCommand = command
	}
}

func Newapp(opts ...AppOption) *App {

	p := &App{
		sdmWrapper:        *sdm.NewSDMClient("sdm"),
		dbPath:            xdg.DataHome,
		dmenuCommand:      Rofi,
		blacklistPatterns: []string{},
	}

	for _, opt := range opts {
		opt(p)
	}

	mustHaveDependencies(p.dmenuCommand)

	db, err := storage.NewStorage(p.account, p.dbPath)

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open database")
	}

	p.db = db

	return p
}

func (p *App) validateAccount() error {
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
	tw := tabwriter.NewWriter(w, 0, 8, 2, '\t', 0)

	for _, ds := range dataSources {
		status := "🔌"

		if ds.Status == "connected" {
			status = "⚡"
		}

		if ds.WebURL != "" {
			status = "🌐"
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

func (p *App) retryCommand(exec func() error) error {
	err := exec()
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
		return p.handleUnauthorized(exec)
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

func (p *App) handleUnauthorized(command func() error) error {
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

func (p *App) handleInvalidCredentials(err error) error {
	notify.Notify("SDM CLI", "Authentication error 🔐", "Invalid credentials", "")
	p.keyring.DeleteSecret(p.account)
	return err
}
