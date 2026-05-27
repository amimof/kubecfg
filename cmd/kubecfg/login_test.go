package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestRunLoginCmdStreamsSubprocessOutputAndImportsAuthInfo(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "target-kubeconfig.yaml")

	originalCfg := cfg
	originalBaseDir := baseDir
	originalStdout := loginStdout
	originalStderr := loginStderr
	t.Cleanup(func() {
		cfg = originalCfg
		baseDir = originalBaseDir
		loginStdout = originalStdout
		loginStderr = originalStderr
	})

	cfg = newLoginCommandTestConfig(targetPath)
	baseDir = filepath.Dir(targetPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	loginStdout = &stdout
	loginStderr = &stderr

	err := runLoginCmd("work", "vgr", "ctx1")
	require.NoError(t, err)
	require.Greater(t, stdout.Len(), 64*1024)
	require.Greater(t, stderr.Len(), 64*1024)
	require.Contains(t, stdout.String(), "stdout-data")
	require.Contains(t, stderr.String(), "stderr-data")

	loaded, err := clientcmd.LoadFromFile(targetPath)
	require.NoError(t, err)

	authInfo, ok := loaded.AuthInfos["login-user"]
	require.True(t, ok)
	require.Equal(t, "imported-token", authInfo.Token)
	require.Equal(t, "login-user", loaded.Contexts["ctx1"].AuthInfo)
}

func TestRunLoginCmdDoesNotMutateCredentialSourceEnv(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "target-kubeconfig.yaml")

	originalCfg := cfg
	originalBaseDir := baseDir
	originalStdout := loginStdout
	originalStderr := loginStderr
	t.Cleanup(func() {
		cfg = originalCfg
		baseDir = originalBaseDir
		loginStdout = originalStdout
		loginStderr = originalStderr
	})

	cfg = newLoginCommandTestConfig(targetPath)
	baseDir = filepath.Dir(targetPath)
	loginStdout = io.Discard
	loginStderr = io.Discard

	compiler := config.NewCompiler(baseDir)
	runtime, err := compiler.Compile(&cfg)
	require.NoError(t, err)

	auth := runtime.Workspace("work").Kubeconfig("vgr").Context("ctx1").AuthInfo
	source, ok := auth.CredentialSource.(*config.RuntimeLoginCredentialSource)
	require.True(t, ok)
	require.NotContains(t, source.Env, "KUBECONFIG")

	err = runLoginCmd("work", "vgr", "ctx1")
	require.NoError(t, err)
	require.NotContains(t, source.Env, "KUBECONFIG")
}

func TestHelperProcessLoginCommand(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		os.Exit(2)
	}

	stdoutPayload := strings.Repeat("stdout-data\n", 10000)
	stderrPayload := strings.Repeat("stderr-data\n", 10000)

	if _, err := io.WriteString(os.Stdout, stdoutPayload); err != nil {
		os.Exit(3)
	}
	if _, err := io.WriteString(os.Stderr, stderrPayload); err != nil {
		os.Exit(4)
	}

	kubeconfig := api.Config{
		Clusters: map[string]*api.Cluster{
			"imported-cluster": {Server: "https://example.com"},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"imported-user": {Token: "imported-token"},
		},
		Contexts: map[string]*api.Context{
			"imported": {
				Cluster:  "imported-cluster",
				AuthInfo: "imported-user",
			},
		},
		CurrentContext: "imported",
	}

	data, err := clientcmd.Write(kubeconfig)
	if err != nil {
		os.Exit(5)
	}

	if err := os.WriteFile(kubeconfigPath, data, 0o600); err != nil {
		os.Exit(6)
	}

	os.Exit(0)
}

func newLoginCommandTestConfig(targetPath string) config.Config {
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
					"login-user": {
						Login: &config.LoginAuth{
							Command:             os.Args[0],
							Args:                []string{"-test.run=TestHelperProcessLoginCommand", "--"},
							Env:                 []string{"GO_WANT_HELPER_PROCESS=1"},
							CopyFromContextName: "imported",
						},
					},
				},
				Contexts: map[string]*config.Context{
					"ctx1": {
						Cluster:  "cluster",
						AuthInfo: "login-user",
					},
				},
			},
		},
	}
}
