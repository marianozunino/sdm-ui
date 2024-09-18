package cmder

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	"github.com/rs/zerolog/log"
)

type (
	CommandOption func(*CommandRunner, *exec.Cmd)
	ErrorParser   func(output string, err error) error
)

type CommandRunner struct {
	Exe          string
	Args         []string
	ErrorParser  ErrorParser
	outputBuffer bytes.Buffer
}

func WithStdin(stdin io.Reader) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cmd.Stdin = stdin
	}
}

func WithOutput(w io.Writer) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cmd.Stdout = io.MultiWriter(w, &cr.outputBuffer)
		cmd.Stderr = io.MultiWriter(w, &cr.outputBuffer)
	}
}

func WithErrorParser(parser ErrorParser) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cr.ErrorParser = parser
	}
}

func WithArgs(args ...string) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cr.Args = args
	}
}

// RunCommand executes a command with the given options
func (r *CommandRunner) RunCommand(opts ...CommandOption) error {
	r.outputBuffer.Reset()

	cmd := exec.Command(r.Exe)

	for _, opt := range opts {
		opt(r, cmd)
	}

	cmd.Args = append(cmd.Args, r.Args...)

	log.Debug().Msgf("Running command: %s %s", r.Exe, strings.Join(r.Args, " "))

	err := cmd.Run()
	if err != nil {
		log.Error().Msgf("Command failed: %s", err)
		if r.ErrorParser != nil {
			return r.ErrorParser(r.outputBuffer.String(), err)
		}
		return err
	}
	return nil
}
