package sdm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/marianozunino/sdm-ui/internal/cmder"
	"github.com/rs/zerolog/log"
)

// DefaultTimeout is the default timeout for SDM operations
const DefaultTimeout = 30 * time.Second

// ErrJSONParsing indicates a failure in parsing JSON output
var ErrJSONParsing = errors.New("failed to parse JSON output")

// SdmReady represents the state of the SDM client
type SdmReady struct {
	Account         *string `json:"account"`
	ListenerRunning bool    `json:"listener_running"`
	StateLoaded     bool    `json:"state_loaded"`
	IsLinked        bool    `json:"is_linked"`
}

// SDMClient provides methods to interact with the SDM CLI
type SDMClient struct {
	CommandRunner *cmder.CommandRunner
	timeout       time.Duration
}

// SDMClientOption defines a function type that modifies SDMClient configuration
type SDMClientOption func(*SDMClient)

// WithTimeout sets a custom timeout for SDM operations
func WithTimeout(timeout time.Duration) SDMClientOption {
	return func(c *SDMClient) {
		c.timeout = timeout
	}
}

// WithErrorParser sets a custom error parser
func WithErrorParser(parser cmder.ErrorParser) SDMClientOption {
	return func(c *SDMClient) {
		c.CommandRunner.ErrorParser = parser
	}
}

// NewSDMClient creates a new SDM client with the specified executable name
func NewSDMClient(exe string, opts ...SDMClientOption) *SDMClient {
	client := &SDMClient{
		CommandRunner: &cmder.CommandRunner{
			Exe:         exe,
			ErrorParser: parseSdmError, // default error parser
		},
		timeout: DefaultTimeout,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// ReadyWithContext checks if the SDM client is ready and returns the state using the provided context
func (s *SDMClient) ReadyWithContext(ctx context.Context) (SdmReady, error) {
	var output strings.Builder

	ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err := s.CommandRunner.RunCommandWithContext(
		ctxWithTimeout,
		cmder.WithArgs("ready"),
		cmder.WithOutput(&output),
	)
	if err != nil {
		return SdmReady{}, fmt.Errorf("ready command failed: %w", err)
	}

	outputStr := output.String()
	log.Debug().Str("output", outputStr).Msg("Ready command output")

	var ready SdmReady
	if err := json.Unmarshal([]byte(outputStr), &ready); err != nil {
		return SdmReady{}, fmt.Errorf("%w: %v", ErrJSONParsing, err)
	}

	return ready, nil
}

// Ready checks if the SDM client is ready and returns the state
func (s *SDMClient) Ready() (SdmReady, error) {
	return s.ReadyWithContext(context.Background())
}

// LogoutWithContext logs out the user from the SDM client using the provided context
func (s *SDMClient) LogoutWithContext(ctx context.Context) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var output strings.Builder
	err := s.CommandRunner.RunCommandWithContext(
		ctxWithTimeout,
		cmder.WithArgs("logout"),
		cmder.WithOutput(&output),
		cmder.WithErrorParser(parseSdmError),
	)
	if err != nil {
		log.Debug().Err(err).Str("output", output.String()).Msg("Logout failed")
		return fmt.Errorf("logout command failed: %w", err)
	}

	log.Debug().Msg("Logout successful")
	return nil
}

// Logout logs out the user from the SDM client
func (s *SDMClient) Logout() error {
	return s.LogoutWithContext(context.Background())
}

// LoginWithContext logs in the user with the provided email and password using the provided context
func (s *SDMClient) LoginWithContext(ctx context.Context, email, password string) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	stdin := strings.NewReader(password + "\n")
	var output strings.Builder

	err := s.CommandRunner.RunCommandWithContext(
		ctxWithTimeout,
		cmder.WithArgs("login", "--email", email),
		cmder.WithStdin(stdin),
		cmder.WithOutput(&output),
		cmder.WithErrorParser(parseSdmError),
	)
	if err != nil {
		log.Debug().
			Err(err).
			Str("email", email).
			Str("output", output.String()).
			Msg("Login failed")
		return fmt.Errorf("login command failed: %w", err)
	}

	log.Debug().Str("email", email).Msg("Login successful")
	return nil
}

// Login logs in the user with the provided email and password
func (s *SDMClient) Login(email, password string) error {
	return s.LoginWithContext(context.Background(), email, password)
}

// StatusWithContext writes the status of the SDM client to the provided writer using the provided context
func (s *SDMClient) StatusWithContext(ctx context.Context, output io.Writer) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	err := s.CommandRunner.RunCommandWithContext(
		ctxWithTimeout,
		cmder.WithArgs("status", "-j"),
		cmder.WithOutput(output),
		cmder.WithErrorParser(parseSdmError),
	)
	if err != nil {
		return fmt.Errorf("status command failed: %w", err)
	}

	return nil
}

// Status writes the status of the SDM client to the provided writer
func (s *SDMClient) Status(output io.Writer) error {
	return s.StatusWithContext(context.Background(), output)
}

// ConnectWithContext connects to the specified data source using the provided context
func (s *SDMClient) ConnectWithContext(ctx context.Context, dataSource string) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var output strings.Builder

	err := s.CommandRunner.RunCommandWithContext(
		ctxWithTimeout,
		cmder.WithArgs("connect", dataSource),
		cmder.WithOutput(&output),
		cmder.WithErrorParser(parseSdmError),
	)
	if err != nil {
		log.Debug().
			Err(err).
			Str("dataSource", dataSource).
			Str("output", output.String()).
			Msg("Connect failed")
		return fmt.Errorf("connect command failed for '%s': %w", dataSource, err)
	}

	log.Debug().Str("dataSource", dataSource).Msg("Connect successful")
	return nil
}

// Connect connects to the specified data source
func (s *SDMClient) Connect(dataSource string) error {
	return s.ConnectWithContext(context.Background(), dataSource)
}
