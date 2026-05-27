package command

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecCommandRunnerHonorsDirAndStdin(t *testing.T) {
	if os.Getenv("GO_WANT_COMMAND_HELPER_PROCESS") == "1" {
		helpCommandRunnerProcess(t)
		return
	}

	runner := NewExecCommandRunner()
	workingDir := t.TempDir()
	var stdout bytes.Buffer

	_, err := runner.Run(context.Background(), CommandSpec{
		Command: os.Args[0],
		Args:    []string{"-test.run=TestExecCommandRunnerHonorsDirAndStdin", "--"},
		Env: map[string]string{
			"GO_WANT_COMMAND_HELPER_PROCESS": "1",
		},
		Dir:    workingDir,
		Stdin:  bytes.NewBufferString("stdin-payload"),
		Stdout: &stdout,
	})
	require.NoError(t, err)
	resolvedWorkingDir, err := filepath.EvalSymlinks(workingDir)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("cwd=%s\nstdin=stdin-payload\n", resolvedWorkingDir), stdout.String())
}

func helpCommandRunnerProcess(t *testing.T) {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err)

	stdin, err := io.ReadAll(os.Stdin)
	require.NoError(t, err)

	_, err = fmt.Fprintf(os.Stdout, "cwd=%s\nstdin=%s\n", filepath.Clean(cwd), string(stdin))
	require.NoError(t, err)

	os.Exit(0)
}
