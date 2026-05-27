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

func (s *LoginService) Login(
	ctx context.Context,
	auth *config.RuntimeAuthInfo,
) (*api.AuthInfo, error) {
	if auth == nil {
		return nil, fmt.Errorf("authinfo is nil")
	}

	switch source := auth.CredentialSource.(type) {
	case nil:
		return auth.AuthInfo, nil
	case *config.RuntimeLoginCredentialSource:
		return s.loginWithCommand(ctx, auth, source)
	default:
		return nil, fmt.Errorf("unsupported credential source %T", source)
	}
}

func (s *LoginService) loginWithCommand(ctx context.Context, _ *config.RuntimeAuthInfo, sourceType config.RuntimeCredentialSource) (*api.AuthInfo, error) {
	source := sourceType.(*config.RuntimeLoginCredentialSource)

	// Create temporary file where kubeconfig is written to by the exec command
	tmpFile, err := os.CreateTemp("/tmp", "kubecfg-login")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			panic(err)
		}
		if err := os.Remove(tmpFile.Name()); err != nil {
			panic(err)
		}
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

	if _, ok := kubeconfig.Contexts[source.Import.Context]; !ok {
		return nil, fmt.Errorf("could not import auth info from context %s", source.Import.Context)
	}

	authInfoRef := kubeconfig.Contexts[source.Import.Context].AuthInfo
	authInfo := kubeconfig.AuthInfos[authInfoRef]

	return authInfo, nil
}
