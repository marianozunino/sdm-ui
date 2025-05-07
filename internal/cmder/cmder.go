package cmder

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

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
	timeout      time.Duration
}

// WithStdin sets the command's standard input
func WithStdin(stdin io.Reader) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cmd.Stdin = stdin
	}
}

// WithOutput redirects command output to the given writer and internal buffer
func WithOutput(w io.Writer) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cmd.Stdout = io.MultiWriter(w, &cr.outputBuffer)
		cmd.Stderr = io.MultiWriter(w, &cr.outputBuffer)
	}
}

// WithStdout sets only the command's standard output
func WithStdout(w io.Writer) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		stdoutWriter := io.MultiWriter(w, &cr.outputBuffer)
		cmd.Stdout = stdoutWriter
	}
}

// WithStderr sets only the command's standard error
func WithStderr(w io.Writer) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		stderrWriter := io.MultiWriter(w, &cr.outputBuffer)
		cmd.Stderr = stderrWriter
	}
}

// WithErrorParser sets a custom error parser
func WithErrorParser(parser ErrorParser) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cr.ErrorParser = parser
	}
}

// WithArgs sets the command arguments
func WithArgs(args ...string) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cr.Args = args
	}
}

// WithTimeout sets a timeout for command execution
func WithTimeout(timeout time.Duration) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cr.timeout = timeout
	}
}

// WithEnv sets environment variables for the command
func WithEnv(env []string) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cmd.Env = env
	}
}

// WithDir sets the working directory for the command
func WithDir(dir string) CommandOption {
	return func(cr *CommandRunner, cmd *exec.Cmd) {
		cmd.Dir = dir
	}
}

// RunCommand executes a command with the given options
func (r *CommandRunner) RunCommand(opts ...CommandOption) error {
	r.outputBuffer.Reset()
	cmd := exec.Command(r.Exe)

	for _, opt := range opts {
		opt(r, cmd)
	}

	cmd.Args = append([]string{r.Exe}, r.Args...)

	log.Debug().
		Str("command", r.Exe).
		Strs("args", r.Args).
		Msg("Running command")

	var err error
	if r.timeout > 0 {
		// Run with timeout using context
		ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
		defer cancel()

		cmdWithContext := exec.CommandContext(ctx, r.Exe, r.Args...)
		cmdWithContext.Stdin = cmd.Stdin
		cmdWithContext.Stdout = cmd.Stdout
		cmdWithContext.Stderr = cmd.Stderr
		cmdWithContext.Env = cmd.Env
		cmdWithContext.Dir = cmd.Dir

		err = cmdWithContext.Run()
	} else {
		err = cmd.Run()
	}

	if err != nil {
		output := r.outputBuffer.String()
		log.Error().
			Err(err).
			Str("output", output).
			Msg("Command failed")

		if r.ErrorParser != nil {
			return r.ErrorParser(output, err)
		}

		return fmt.Errorf("command failed: %s: %w", output, err)
	}

	return nil
}

// RunCommandWithContext executes a command with the given context and options
func (r *CommandRunner) RunCommandWithContext(ctx context.Context, opts ...CommandOption) error {
	r.outputBuffer.Reset()

	// Create base command
	cmd := exec.CommandContext(ctx, r.Exe)

	// Apply options
	for _, opt := range opts {
		opt(r, cmd)
	}

	// Set arguments
	cmd.Args = append([]string{r.Exe}, r.Args...)

	log.Debug().
		Str("command", r.Exe).
		Strs("args", r.Args).
		Msg("Running command with context")

	// Execute the command
	err := cmd.Run()
	if err != nil {
		output := r.outputBuffer.String()
		log.Error().
			Err(err).
			Str("output", output).
			Msg("Command failed")

		if r.ErrorParser != nil {
			return r.ErrorParser(output, err)
		}

		return fmt.Errorf("command failed: %s: %w", output, err)
	}

	return nil
}

// GetOutput returns the command output as a string
func (r *CommandRunner) GetOutput() string {
	return r.outputBuffer.String()
}

// ResetBuffer clears the output buffer
func (r *CommandRunner) ResetBuffer() {
	r.outputBuffer.Reset()
}
