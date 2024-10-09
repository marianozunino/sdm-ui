package sdm

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/marianozunino/sdm-ui/internal/cmder"
)

type SdmReady struct {
	Account         *string `json:"account"`
	ListenerRunning bool    `json:"listener_running"`
	StateLoaded     bool    `json:"state_loaded"`
	IsLinked        bool    `json:"is_linked"`
}

type SDMClient struct {
	CommandRunner *cmder.CommandRunner
}

func NewSDMClient(exe string) *SDMClient {
	return &SDMClient{
		CommandRunner: &cmder.CommandRunner{
			Exe:         exe,
			ErrorParser: parseSdmError, // default error parser
		},
	}
}

// Ready checks if the SDM client is ready and returns the state.
func (s *SDMClient) Ready() (SdmReady, error) {
	var output strings.Builder
	err := s.CommandRunner.RunCommand(
		cmder.WithArgs("ready"),
		cmder.WithOutput(&output),
	)
	if err != nil {
		return SdmReady{}, err
	}
	var ready SdmReady
	if err := json.Unmarshal([]byte(output.String()), &ready); err != nil {
		return SdmReady{}, err
	}
	return ready, nil
}

// Logout logs out the user from the SDM client.
func (s *SDMClient) Logout() error {
	return s.CommandRunner.RunCommand(
		cmder.WithArgs("logout"),
		cmder.WithOutput(io.Discard),
		cmder.WithErrorParser(parseSdmError),
	)
}

// Login logs in the user with the provided email and password.
func (s *SDMClient) Login(email, password string) error {
	stdin := strings.NewReader(password + "\n")
	return s.CommandRunner.RunCommand(
		cmder.WithArgs("login", "--email", email),
		cmder.WithStdin(stdin),
		cmder.WithOutput(io.Discard),
		cmder.WithErrorParser(parseSdmError),
	)
}

// Status writes the status of the SDM client to the provided writer.
func (s *SDMClient) Status(output io.Writer) error {
	return s.CommandRunner.RunCommand(
		cmder.WithArgs("status", "-j"),
		cmder.WithOutput(output),
		cmder.WithErrorParser(parseSdmError),
	)
}

// Connect connects to the specified data source.
func (s *SDMClient) Connect(dataSource string) error {
	return s.CommandRunner.RunCommand(
		cmder.WithArgs("connect", dataSource),
		cmder.WithOutput(io.Discard),
		cmder.WithErrorParser(parseSdmError),
	)
}
