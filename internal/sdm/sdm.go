package sdm

import (
	"encoding/json"
	"io"
	"os/exec"
	"strings"
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
	msg  string
}

func (e SDMError) Error() string {
	return e.msg
}

type SdmReady struct {
	Account         *string `json:"account"`
	ListenerRunning bool    `json:"listener_running"`
	StateLoaded     bool    `json:"state_loaded"`
	IsLinked        bool    `json:"is_linked"`
}

func (s *SDMClient) Ready() (SdmReady, error) {
	cmd := exec.Command(s.Exe, "ready")
	stdout, err := cmd.CombinedOutput()
	decoder := json.NewDecoder(strings.NewReader(string(stdout)))
	var ready SdmReady
	err = decoder.Decode(&ready)
	if err != nil {
		panic(err)
	}
	return ready, nil
}

func (s *SDMClient) Logout() error {
	cmd := exec.Command(s.Exe, "logout")
	stdout, err := cmd.CombinedOutput()
	return parseSdmError(string(stdout), err)
}

func (s *SDMClient) Login(email string, password string) error {
	cmd := exec.Command(s.Exe, "login", "--email", email)
	cmd.Stdin = strings.NewReader(password + "\n")
	output, err := cmd.CombinedOutput()
	return parseSdmError(string(output), err)
}

func (s *SDMClient) Status(w io.Writer) error {
	cmd := exec.Command(s.Exe, "status")
	output, err := cmd.CombinedOutput()
	w.Write(output)
	return parseSdmError(string(output), err)
}

func (s *SDMClient) Connect(dataSource string) error {
	cmd := exec.Command(s.Exe, "connect", dataSource)
	stdout, err := cmd.CombinedOutput()
	return parseSdmError(string(stdout), err)
}

func parseSdmError(output string, err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(string(output), "You are not authenticated") {
		return SDMError{Code: Unauthorized, msg: string(output)}
	}
	if strings.Contains(string(output), "access denied") {
		return SDMError{Code: InvalidCredentials, msg: string(output)}
	}
	if strings.Contains(string(output), "doesn't have a strongDM account") {
		return SDMError{Code: InvalidCredentials, msg: string(output)}
	}
	if strings.Contains(string(output), "Cannot find datasource named") {
		return SDMError{Code: ResourceNotFound, msg: string(output)}
	}
	return SDMError{Code: Unknown, msg: string(output)}
}
