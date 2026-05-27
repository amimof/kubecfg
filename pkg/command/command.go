package command

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, spec CommandSpec) (*CommandResult, error)
}

type CommandSpec struct {
	Command string
	Args    []string

	Env map[string]string
	Dir string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Redact []string
}

type CommandResult struct {
	ExitCode int

	Stdout []byte
	Stderr []byte
}

type ExecCommandRunner struct{}

func NewExecCommandRunner() *ExecCommandRunner {
	return &ExecCommandRunner{}
}

func (c *ExecCommandRunner) Run(ctx context.Context, spec CommandSpec) (*CommandResult, error) {
	if strings.TrimSpace(spec.Command) == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Create command
	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...)
	cmd.Env = mergeEnv(os.Environ(), spec.Env)
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr

	if spec.Stdout == nil {
		cmd.Stdout = &stdout
	}

	if spec.Stderr == nil {
		cmd.Stderr = &stderr
	}

	// Run command and stream output to avoid pipe deadlocks.
	err := cmd.Run()

	result := &CommandResult{
		ExitCode: exitCode(err),
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
	}

	if err != nil {
		return result, &CommandError{
			Command:  spec.Command,
			Args:     spec.Args,
			ExitCode: result.ExitCode,
			Stdout:   redactBytes(result.Stdout, spec.Redact),
			Stderr:   redactBytes(result.Stderr, spec.Redact),
			Err:      err,
		}
	}

	return result, nil
}

type CommandError struct {
	Command  string
	Args     []string
	ExitCode int

	Stdout []byte
	Stderr []byte

	Err error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf(
		"command %q failed with exit code %d: %s",
		e.Command,
		e.ExitCode,
		strings.TrimSpace(string(e.Stderr)),
	)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

func redactBytes(data []byte, secrets []string) []byte {
	if len(data) == 0 || len(secrets) == 0 {
		return data
	}

	s := string(data)

	for _, secret := range secrets {
		if secret == "" {
			continue
		}

		s = strings.ReplaceAll(s, secret, "REDACTED")
	}

	return []byte(s)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}

	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return exitErr.ExitCode()
	}

	return -1
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}

	env := make(map[string]string, len(base)+len(extra))

	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}

		env[key] = value
	}

	maps.Copy(env, extra)

	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}

	sort.Strings(out)

	return out
}
