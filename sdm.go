package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/ncruces/zenity"
	"github.com/tmc/keyring"
)

const KEYRING_KEY = "sdm-credential"

func syncStatuses(w io.Writer) error {
	cmd := exec.Command("sdm", "status")
	stdout, err := cmd.Output()
	if err != nil {
		return err
	}
	w.Write(stdout)

	return nil
}

func connectToDataSource(dataSource string) error {
	cmd := exec.Command("sdm", "connect", dataSource)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(stdout), "You are not atuhenticated") {
			printDebug("[login] Unauthenticated, please re-authenticate")
			return sdmError{code: unauthorized, msg: string(stdout)}
		}
		return err
	}
	return nil
}

type sdmErrorCode int

const (
	unauthorized sdmErrorCode = 1
	invalidCreds sdmErrorCode = 2
	otherError   sdmErrorCode = 3
)

type sdmError struct {
	code sdmErrorCode
	msg  string
}

func (e sdmError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.msg)
}

func authenticate(email string, password string) error {
	cmd1 := exec.Command("echo", password)
	cmd2 := exec.Command("sdm", "login", "--email", email)

	pipe, err := cmd1.StdoutPipe()
	if err != nil {
		printDebug("[login] Failed to authenticate, unknown error")
		return sdmError{code: otherError, msg: err.Error()}
	}

	cmd2.Stdin = pipe
	if err := cmd1.Start(); err != nil {
		printDebug("[login] Failed to authenticate, unknown error")
		return sdmError{code: otherError, msg: err.Error()}
	}

	if output, _ := cmd2.CombinedOutput(); err != nil {
		if strings.Contains(string(output), "access denied") {
			printDebug("[login] Failed to authenticate, access denied")
			return sdmError{code: invalidCreds, msg: string(output)}
		}
		if strings.Contains(string(output), "doesn't have a strongDM account") {
			printDebug("[login] Failed to authenticate, doesn't have a strongDM account")
			return sdmError{code: invalidCreds, msg: string(output)}
		}
		printDebug("[login] Failed to authenticate, unknown error")
		return sdmError{code: otherError, msg: string(output)}
	}
	return nil
}

func getSecretFromKeyring(email string) (string, error) {
	return keyring.Get(KEYRING_KEY, email)
}

func setSecretToKeyring(email string, secret string) error {
	return keyring.Set(KEYRING_KEY, email, secret)
}

func askForPassword() (string, error) {
	printDebug("[login] Prompting for password...")
	_, pwd, err := zenity.Password(zenity.Title("Type your SDM password"))
	return pwd, err
}

func readOrAskForPassword(email string) (string, error) {
	password, err := getSecretFromKeyring(email)
	if err != nil {
		password, err = askForPassword()

		if err != nil {
			return "", err
		}

		if err := setSecretToKeyring(email, password); err != nil {
			return "", err
		}
	}
	return password, nil
}
