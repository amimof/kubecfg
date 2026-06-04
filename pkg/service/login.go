package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"

	"github.com/amimof/kubecfg/pkg/command"
	"github.com/amimof/kubecfg/pkg/config"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type LoginService struct {
	// StateStore CredentialStateStore
	Runner command.CommandRunner
	Stdout io.Writer
	Stderr io.Writer
}

func (s *LoginService) Login(ctx context.Context, rkc *config.RuntimeKubeconfig) error {
	if rkc == nil {
		return fmt.Errorf("runtime kubeconfig is nil")
	}

	for _, source := range rkc.LoginSources {
		imported, err := s.loginWithCommand(ctx, source)
		if err != nil {
			return fmt.Errorf("login source %q: %w", source.Name, err)
		}
		source.ImportedConfig = imported
	}

	return nil
}

func (s *LoginService) loginWithCommand(ctx context.Context, source *config.RuntimeLoginSource) (*api.Config, error) {
	// Create temporary file where kubeconfig is written to by the exec command
	tmpFile, err := os.CreateTemp("/tmp", "kubecfg-login")
	if err != nil {
		return nil, fmt.Errorf("create temporary kubeconfig: %w", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			panic(err)
		}
		// if err := os.Remove(tmpFile.Name()); err != nil {
		// 	panic(err)
		// }
	}()

	// Add var so that login uses temporary kubeconfig file during login
	env := make(map[string]string, len(source.Env)+1)
	maps.Copy(env, source.Env)
	env["KUBECONFIG"] = tmpFile.Name()

	stdoutWriter, _ := teeWriter(s.Stdout)
	stderrWriter, stderrBuf := teeWriter(s.Stderr)

	_, err = s.Runner.Run(ctx, command.CommandSpec{
		Command: source.Command,
		Args:    source.Args,
		Env:     env,
		Dir:     "/tmp",
		Stdout:  stdoutWriter,
		Stderr:  stderrWriter,
	})
	if err != nil {
		return nil, wrapLoginCommandError(source.Command, err, stderrBuf.String())
	}

	// Read temporary kubeconfig to extract token
	kubeconfig, err := clientcmd.LoadFromFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("load generated kubeconfig: %w", err)
	}

	return kubeconfig, nil
}

func teeWriter(w io.Writer) (io.Writer, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	if w == nil {
		return buf, buf
	}

	return io.MultiWriter(w, buf), buf
}

func wrapLoginCommandError(commandName string, err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("run command %q: %w", commandName, err)
	}

	return fmt.Errorf("run command %q: %w: %s", commandName, err, stderr)
}
