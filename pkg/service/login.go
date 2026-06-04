package service

import (
	"context"
	"fmt"
	"io"
	"maps"
	"os"

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
			return err
		}
		source.ImportedConfig = imported
	}

	return nil
}

func (s *LoginService) loginWithCommand(ctx context.Context, source *config.RuntimeLoginSource) (*api.Config, error) {
	// Create temporary file where kubeconfig is written to by the exec command
	tmpFile, err := os.CreateTemp("/tmp", "kubecfg-login")
	if err != nil {
		return nil, err
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

	_, err = s.Runner.Run(ctx, command.CommandSpec{
		Command: source.Command,
		Args:    source.Args,
		Env:     env,
		Dir:     "/tmp",
		Stdout:  s.Stdout,
		Stderr:  s.Stderr,
	})
	if err != nil {
		return nil, err
	}

	// Read temporary kubeconfig to extract token
	kubeconfig, err := clientcmd.LoadFromFile(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	return kubeconfig, nil
}
