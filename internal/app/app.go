package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/adrg/xdg"
	"github.com/marianozunino/sdm-ui/internal/libsecret"
	"github.com/marianozunino/sdm-ui/internal/logger"
	"github.com/marianozunino/sdm-ui/internal/sdm"
	"github.com/marianozunino/sdm-ui/internal/storage"
	"github.com/martinlindhe/notify"
	"github.com/rs/zerolog/log"
)

// ErrResourceNotFound indicates that a requested resource was not found
var ErrResourceNotFound = errors.New("resource not found")

// App represents the main application structure
type App struct {
	account string

	db              *storage.Storage
	dbPath          string
	keyring         libsecret.Keyring
	sdmWrapper      sdm.SDMClient
	dmenuCommand    DMenuCommand
	passwordCommand PasswordCommand

	blacklistPatterns []string
	context           context.Context
	timeout           time.Duration
}

// AppOption defines a function type that modifies App configuration
type AppOption func(*App)

// WithAccount sets the account for the App
func WithAccount(account string) AppOption {
	return func(p *App) {
		log.Debug().Str("account", account).Msg("Using account")
		p.account = account
	}
}

// WithVerbose configures verbose logging
func WithVerbose(verbose bool) AppOption {
	logger.ConfigureLogger(verbose)
	return func(p *App) {}
}

// WithDbPath sets the database path
func WithDbPath(dbPath string) AppOption {
	return func(p *App) {
		p.dbPath = dbPath
	}
}

// WithBlacklist sets patterns for blacklisting resources
func WithBlacklist(patterns []string) AppOption {
	return func(p *App) {
		p.blacklistPatterns = patterns
	}
}

// WithCommand sets the menu command to use
func WithCommand(command DMenuCommand) AppOption {
	return func(p *App) {
		p.dmenuCommand = command
	}
}

// WithPasswordCommand sets the password command to use
func WithPasswordCommand(command PasswordCommand) AppOption {
	return func(p *App) {
		p.passwordCommand = command
	}
}

// WithTimeout sets a timeout for operations
func WithTimeout(timeout time.Duration) AppOption {
	return func(p *App) {
		p.timeout = timeout
	}
}

// WithContext sets a context for the application
func WithContext(ctx context.Context) AppOption {
	return func(p *App) {
		p.context = ctx
	}
}

// NewApp creates a new application instance with the provided options
func NewApp(opts ...AppOption) (*App, error) {
	p := &App{
		sdmWrapper:        *sdm.NewSDMClient("sdm"),
		dbPath:            xdg.DataHome,
		dmenuCommand:      DMenuCommandRofi,
		blacklistPatterns: []string{},
		passwordCommand:   PasswordCommandZenity,
		context:           context.Background(),
		timeout:           30 * time.Second, // Default timeout
	}

	for _, opt := range opts {
		opt(p)
	}

	if err := p.mustHaveDependencies(); err != nil {
		return nil, fmt.Errorf("dependency check failed: %w", err)
	}

	db, err := storage.NewStorage(p.account, p.dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	p.db = db

	return p, nil
}

// Close closes all resources held by the App
func (p *App) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// ValidateAccount ensures the user is authenticated with the correct account
func (p *App) ValidateAccount() error {
	ctx, cancel := context.WithTimeout(p.context, p.timeout)
	defer cancel()

	status, err := p.sdmWrapper.ReadyWithContext(ctx)
	if err != nil {
		return fmt.Errorf("ready check failed: %w", err)
	}

	if status.Account != nil && *status.Account != p.account {
		log.Debug().
			Str("current", *status.Account).
			Str("expected", p.account).
			Msg("Logged in with a different account, logging out")

		if err := p.sdmWrapper.LogoutWithContext(ctx); err != nil {
			var sdmErr sdm.SDMError
			if errors.As(err, &sdmErr) && sdmErr.Code == sdm.Unauthorized {
				// Already logged out
				return nil
			}
			return fmt.Errorf("failed to logout: %w", err)
		}
	}

	return nil
}

// PrintDataSources formats and writes data sources to the provided writer
func (p *App) PrintDataSources(dataSources []storage.DataSource, w io.Writer, withHeaders bool) {
	const format = "%v\t%v\t%v\n"
	tw := tabwriter.NewWriter(w, 0, 8, 2, '\t', 0)

	// Write header
	if withHeaders {
		fmt.Fprintf(tw, format, "NAME", "ADDRESS", "STATUS")
		fmt.Fprintf(tw, format, "----", "-------", "------")
	}

	for _, ds := range dataSources {
		status := "🔌"

		if ds.Status == "connected" {
			status = "⚡"
		}

		if ds.WebURL != "" {
			status = "🌐"
		}

		fmt.Fprintf(tw, format, ds.Name, Ellipsize(ds.Address, 20), status)
	}
	tw.Flush()
}

// Ellipsize truncates a string to maxLen and adds ellipsis if necessary
func Ellipsize(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// RetryCommand executes the provided function and handles common errors
func (p *App) RetryCommand(exec func() error) error {
	err := exec()
	if err == nil {
		return nil
	}

	var sdmErr sdm.SDMError
	if !errors.As(err, &sdmErr) {
		notify.Notify("SDM CLI", "❗Unexpected error", err.Error(), "")
		return fmt.Errorf("unexpected error: %w", err)
	}

	switch sdmErr.Code {
	case sdm.Unauthorized:
		return p.handleUnauthorized(exec)
	case sdm.InvalidCredentials:
		return p.handleInvalidCredentials(sdmErr)
	case sdm.ResourceNotFound:
		notify.Notify("SDM CLI", "🔐 Resource not found", sdmErr.Error(), "")
		return fmt.Errorf("%w: %v", ErrResourceNotFound, sdmErr)
	default:
		notify.Notify("SDM CLI", "🔐 Error", sdmErr.Error(), "")
		return fmt.Errorf("command error: %w", sdmErr)
	}
}

// HandleUnauthorized handles unauthorized errors by re-authenticating
func (p *App) handleUnauthorized(command func() error) error {
	notify.Notify("SDM CLI", "🔐 Authenticating...", "", "")

	password, err := p.retrievePassword()
	if err != nil {
		notify.Notify("SDM CLI", "🔐 Authentication error", err.Error(), "")
		return fmt.Errorf("failed to retrieve password: %w", err)
	}

	log.Debug().Msg("Logging in...")

	ctx, cancel := context.WithTimeout(p.context, p.timeout)
	defer cancel()

	if err := p.sdmWrapper.LoginWithContext(ctx, p.account, password); err != nil {
		p.keyring.DeleteSecret(p.account)
		notify.Notify("SDM CLI", "🔐 Authentication error", err.Error(), "")
		return fmt.Errorf("login failed: %w", err)
	}

	log.Debug().Msg("Login successful")
	return command()
}

// HandleInvalidCredentials handles invalid credential errors
func (p *App) handleInvalidCredentials(err sdm.SDMError) error {
	notify.Notify("SDM CLI", "🔐 Authentication error", "Invalid credentials", "")
	p.keyring.DeleteSecret(p.account)
	return fmt.Errorf("invalid credentials: %w", err)
}
