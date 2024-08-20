package sdm

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

type SDMClient struct {
	Exe string
}

type SDMErrorCode int

const (
	Unauthorized SDMErrorCode = iota
	InvalidCredentials
	Unknown
	ResourceNotFound
)

type SDMError struct {
	Code SDMErrorCode
	Msg  string
}

func (e SDMError) Error() string {
	return e.Msg
}

type SdmReady struct {
	Account         *string `json:"account"`
	ListenerRunning bool    `json:"listener_running"`
	StateLoaded     bool    `json:"state_loaded"`
	IsLinked        bool    `json:"is_linked"`
}

// RunCommand executes a command, logs the SDM version, and returns the command output or an error.
func (s *SDMClient) RunCommand(args ...string) (string, error) {
	log.Debug().Msgf("Running command: %s %s", s.Exe, strings.Join(args, " "))

	// Execute the command
	cmd := exec.Command(s.Exe, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error().Msgf("Command failed: %s (%s)", output, err)
	}
	return string(output), err
}

// Ready checks if the SDM client is ready and returns the state.
func (s *SDMClient) Ready() (SdmReady, error) {
	output, err := s.RunCommand("ready")

	fmt.Printf("output: %s\n", output)
	fmt.Printf("output: %s\n", err)

	if err != nil {
		return SdmReady{}, parseSdmError(output, err)
	}

	var ready SdmReady
	if err := json.Unmarshal([]byte(output), &ready); err != nil {
		return SdmReady{}, err
	}

	return ready, nil
}

// Logout logs out the user from the SDM client.
func (s *SDMClient) Logout() error {
	output, err := s.RunCommand("logout")
	return parseSdmError(output, err)
}

// Login logs in the user with the provided email and password.
func (s *SDMClient) Login(email, password string) error {
	cmd := exec.Command(s.Exe, "login", "--email", email)
	cmd.Stdin = strings.NewReader(password + "\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug().Msg(string(err.Error()))
	}
	return parseSdmError(string(output), err)
}

// Status writes the status of the SDM client to the provided writer.
func (s *SDMClient) Status(w io.Writer) error {
	output, err := s.RunCommand("status", "-j")
	if _, writeErr := w.Write([]byte(output)); writeErr != nil {
		return writeErr
	}
	return parseSdmError(output, err)
}

// Connect connects to the specified data source.
func (s *SDMClient) Connect(dataSource string) error {
	output, err := s.RunCommand("connect", dataSource)
	return parseSdmError(output, err)
}

// parseSdmError parses the output and error to return an SDMError with the appropriate code.
func parseSdmError(output string, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case strings.Contains(output, "You are not authenticated"):
		return SDMError{Code: Unauthorized, Msg: output}
	case strings.Contains(output, "Cannot find datasource named"):
		return SDMError{Code: ResourceNotFound, Msg: output}
	case strings.Contains(output, "access denied"):
		return SDMError{Code: InvalidCredentials, Msg: output}
	case strings.Contains(output, "Invalid credentials"):
		return SDMError{Code: InvalidCredentials, Msg: output}
	default:
		return SDMError{Code: Unknown, Msg: output}
	}
}
