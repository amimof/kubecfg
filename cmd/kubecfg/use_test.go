package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"filippo.io/age"
	"github.com/amimof/kubecfg/pkg/config"
	decryptpkg "github.com/amimof/kubecfg/pkg/decrypt"
	fzf "github.com/junegunn/fzf/src"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestPickContextReturnsSelection(t *testing.T) {
	t.Setenv("FZF_DEFAULT_OPTS", "")
	t.Setenv("FZF_DEFAULT_OPTS_FILE", "")

	originalFzfRun := fzfRun
	t.Cleanup(func() {
		fzfRun = originalFzfRun
	})

	fzfRun = func(options *fzf.Options) (int, error) {
		var inputs []string
		for input := range options.Input {
			inputs = append(inputs, input)
		}

		require.Contains(t, inputs, "prod/api")
		options.Output <- "prod/api"
		return fzf.ExitOk, nil
	}

	runtimeConfig := &config.RuntimeConfig{
		Workspaces: map[string]*config.RuntimeWorkspace{
			"dev": {
				Name: "dev",
				Kubeconfigs: map[string]*config.RuntimeKubeconfig{
					"api": {Name: "api"},
				},
			},
			"prod": {
				Name: "prod",
				Kubeconfigs: map[string]*config.RuntimeKubeconfig{
					"api": {Name: "api"},
				},
			},
		},
	}

	workspace, selected, err := pickContext(runtimeConfig, "")
	require.NoError(t, err)
	require.Equal(t, "prod", workspace)
	require.Equal(t, "api", selected)
}

func TestPickContextNoSelection(t *testing.T) {
	t.Setenv("FZF_DEFAULT_OPTS", "")
	t.Setenv("FZF_DEFAULT_OPTS_FILE", "")

	runtimeConfig := &config.RuntimeConfig{
		Workspaces: map[string]*config.RuntimeWorkspace{
			"prod": {
				Name: "prod",
				Kubeconfigs: map[string]*config.RuntimeKubeconfig{
					"api": {Name: "api"},
				},
			},
		},
	}

	tests := []struct {
		name string
		code int
	}{
		{name: "interrupt", code: fzf.ExitInterrupt},
		{name: "no-match", code: fzf.ExitNoMatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalFzfRun := fzfRun
			t.Cleanup(func() {
				fzfRun = originalFzfRun
			})

			fzfRun = func(options *fzf.Options) (int, error) {
				for range options.Input {
				}
				return tt.code, nil
			}

			_, _, err := pickContext(runtimeConfig, "")
			require.ErrorIs(t, err, errNoSelection)
		})
	}
}

func TestRunUseCmdFzfNoSelectionIsNoOp(t *testing.T) {
	t.Setenv("FZF_DEFAULT_OPTS", "")
	t.Setenv("FZF_DEFAULT_OPTS_FILE", "")

	targetPath := filepath.Join(t.TempDir(), "target-kubeconfig.yaml")

	originalCfg := cfg
	originalBaseDir := baseDir
	originalFzfRun := fzfRun
	t.Cleanup(func() {
		cfg = originalCfg
		baseDir = originalBaseDir
		fzfRun = originalFzfRun
	})

	cfg = newUseCommandTestConfig(targetPath)
	baseDir = filepath.Dir(targetPath)
	fzfRun = func(options *fzf.Options) (int, error) {
		for range options.Input {
		}
		return fzf.ExitNoMatch, nil
	}

	err := runUseCmdFzf("", false, "")
	require.NoError(t, err)

	_, err = os.Stat(targetPath)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRunUseCmdFzfUpdatesActiveConfigSymlink(t *testing.T) {
	t.Setenv("FZF_DEFAULT_OPTS", "")
	t.Setenv("FZF_DEFAULT_OPTS_FILE", "")

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "target-kubeconfig.yaml")

	originalCfg := cfg
	originalBaseDir := baseDir
	originalFzfRun := fzfRun
	t.Cleanup(func() {
		cfg = originalCfg
		baseDir = originalBaseDir
		fzfRun = originalFzfRun
	})

	cfg = newUseCommandTestConfig(targetPath)
	baseDir = tmpDir
	fzfRun = func(options *fzf.Options) (int, error) {
		for range options.Input {
		}
		options.Output <- "work/vgr"
		return fzf.ExitOk, nil
	}

	err := runUseCmdFzf("", false, "")
	require.NoError(t, err)

	linkPath := filepath.Join(baseDir, "config")
	linkedTo, err := os.Readlink(linkPath)
	require.NoError(t, err)
	require.Equal(t, targetPath, linkedTo)
	_, err = os.Stat(targetPath)
	require.NoError(t, err)
}

func TestRunUseCmdDecryptsEncryptedTokenWithIdentityFile(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "target-kubeconfig.yaml")
	identityFile, encryptedToken := writeAgeIdentityAndEncryptedToken(t, "command-token")

	originalCfg := cfg
	originalBaseDir := baseDir
	t.Cleanup(func() {
		cfg = originalCfg
		baseDir = originalBaseDir
	})

	cfg = newEncryptedUseCommandTestConfig(targetPath, encryptedToken)
	baseDir = filepath.Dir(targetPath)

	err := runUseCmd("work", "vgr", true, identityFile)
	require.NoError(t, err)

	contents, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Contains(t, string(contents), "token: command-token")
	require.NotContains(t, string(contents), encryptedToken)
}

func TestRunUseCmdFzfDecryptsEncryptedTokenWithIdentityFile(t *testing.T) {
	t.Setenv("FZF_DEFAULT_OPTS", "")
	t.Setenv("FZF_DEFAULT_OPTS_FILE", "")

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "target-kubeconfig.yaml")
	identityFile, encryptedToken := writeAgeIdentityAndEncryptedToken(t, "fzf-token")

	originalCfg := cfg
	originalBaseDir := baseDir
	originalFzfRun := fzfRun
	t.Cleanup(func() {
		cfg = originalCfg
		baseDir = originalBaseDir
		fzfRun = originalFzfRun
	})

	cfg = newEncryptedUseCommandTestConfig(targetPath, encryptedToken)
	baseDir = tmpDir
	fzfRun = func(options *fzf.Options) (int, error) {
		for range options.Input {
		}
		options.Output <- "work/vgr"
		return fzf.ExitOk, nil
	}

	err := runUseCmdFzf("", true, identityFile)
	require.NoError(t, err)

	contents, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Contains(t, string(contents), "token: fzf-token")
}

func TestWriteKubeconfigSetsSecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}

	kubeconfig := newWriteKubeconfigTestConfig()

	t.Run("new file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")

		err := writeKubeconfig(path, kubeconfig)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), filePerms(t, path))
	})

	t.Run("existing file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.yaml")
		err := os.WriteFile(path, []byte("stale"), 0o644)
		require.NoError(t, err)
		require.NoError(t, os.Chmod(path, 0o644))

		err = writeKubeconfig(path, kubeconfig)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), filePerms(t, path))
	})
}

func filePerms(t *testing.T, path string) os.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.Mode().Perm()
}

func newWriteKubeconfigTestConfig() api.Config {
	return api.Config{
		Clusters: map[string]*api.Cluster{
			"cluster": {Server: "https://example.com"},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"user": {},
		},
		Contexts: map[string]*api.Context{
			"context": {
				Cluster:  "cluster",
				AuthInfo: "user",
			},
		},
		CurrentContext: "context",
	}
}

func newUseCommandTestConfig(targetPath string) config.Config {
	return config.Config{
		Version: "v1",
		Workspaces: map[string]*config.Workspace{
			"work": {
				Kubeconfigs: []string{"vgr"},
			},
		},
		Kubeconfigs: map[string]*config.Kubeconfig{
			"vgr": {
				Path: targetPath,
				Clusters: map[string]*config.Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*config.AuthInfo{
					"user": {},
				},
				Contexts: map[string]*config.Context{
					"context": {
						Cluster:  "cluster",
						AuthInfo: "user",
					},
				},
			},
		},
	}
}

func newEncryptedUseCommandTestConfig(targetPath, encryptedToken string) config.Config {
	cfg := newUseCommandTestConfig(targetPath)
	cfg.Kubeconfigs["vgr"].AuthInfos["user"] = &config.AuthInfo{EncryptedToken: encryptedToken}
	return cfg
}

func writeAgeIdentityAndEncryptedToken(t *testing.T, plaintext string) (string, string) {
	t.Helper()

	identity, err := age.GenerateX25519Identity()
	require.NoError(t, err)

	encryptor, err := decryptpkg.NewAgeEncryptor(identity.Recipient())
	require.NoError(t, err)

	encrypted, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	identityFile := filepath.Join(t.TempDir(), "identity.txt")
	identityContents := fmt.Sprintf("# created: now\n# public key: %s\n%s\n", identity.Recipient(), identity.String())
	require.NoError(t, os.WriteFile(identityFile, []byte(identityContents), 0o600))

	return identityFile, encrypted
}
