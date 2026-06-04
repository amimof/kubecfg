package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestRunLoginCmdImportsReferencedContext(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "target-kubeconfig.yaml")

	originalCfg := cfg
	originalStdout := loginStdout
	originalStderr := loginStderr
	t.Cleanup(func() {
		cfg = originalCfg
		loginStdout = originalStdout
		loginStderr = originalStderr
	})

	cfg = newImportedLoginCommandTestConfig(targetPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	loginStdout = &stdout
	loginStderr = &stderr

	err := runLoginCmd("work", "vgr", "ctx1", "")
	require.NoError(t, err)
	require.Greater(t, stdout.Len(), 64*1024)
	require.Greater(t, stderr.Len(), 64*1024)

	loaded, err := clientcmd.LoadFromFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, "imported-cluster", loaded.Contexts["ctx1"].Cluster)
	require.Equal(t, "utbildning-dev", loaded.Contexts["ctx1"].AuthInfo)
	require.Equal(t, "https://example.com", loaded.Clusters["imported-cluster"].Server)
	require.Equal(t, "imported-token", loaded.AuthInfos["utbildning-dev"].Token)
	require.Equal(t, "default", loaded.Contexts["ctx1"].Namespace)
	_, hasLocalCluster := loaded.Clusters["cluster"]
	require.False(t, hasLocalCluster)
}

func TestRunLoginCmdDoesNotMutateLoginSourceEnv(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "target-kubeconfig.yaml")

	originalCfg := cfg
	originalStdout := loginStdout
	originalStderr := loginStderr
	t.Cleanup(func() {
		cfg = originalCfg
		loginStdout = originalStdout
		loginStderr = originalStderr
	})

	cfg = newImportedLoginCommandTestConfig(targetPath)
	loginStdout = &bytes.Buffer{}
	loginStderr = &bytes.Buffer{}

	compiler := config.NewCompiler()
	runtime, err := compiler.Compile(&cfg)
	require.NoError(t, err)

	source := runtime.Workspace("work").Kubeconfig("vgr").LoginSources["shared"]
	require.NotNil(t, source)
	require.NotContains(t, source.Env, "KUBECONFIG")

	err = runLoginCmd("work", "vgr", "ctx1", "")
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

	stdoutPayload := bytes.Repeat([]byte("stdout-data\n"), 10000)
	stderrPayload := bytes.Repeat([]byte("stderr-data\n"), 10000)

	if _, err := os.Stdout.Write(stdoutPayload); err != nil {
		os.Exit(3)
	}
	if _, err := os.Stderr.Write(stderrPayload); err != nil {
		os.Exit(4)
	}

	kubeconfig := api.Config{
		Clusters: map[string]*api.Cluster{
			"imported-cluster": {Server: "https://example.com"},
		},
		AuthInfos: map[string]*api.AuthInfo{
			"imported-user":  {Token: "imported-token"},
			"utbildning-dev": {Token: "imported-token"},
		},
		Contexts: map[string]*api.Context{
			"utbildning-dev": {
				Cluster:  "imported-cluster",
				AuthInfo: "imported-user",
			},
		},
		CurrentContext: "utbildning-dev",
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

func newImportedLoginCommandTestConfig(targetPath string) config.Config {
	return config.Config{
		Version: "v1",
		BaseDir: filepath.Dir(targetPath),
		Workspaces: map[string]*config.Workspace{
			"work": {
				Kubeconfigs: []string{"vgr"},
			},
		},
		Kubeconfigs: map[string]*config.Kubeconfig{
			"vgr": {
				Path: targetPath,
				LoginSources: map[string]*config.LoginSource{
					"shared": {
						Command: os.Args[0],
						Args:    []string{"-test.run=TestHelperProcessLoginCommand", "--"},
						Env:     []string{"GO_WANT_HELPER_PROCESS=1"},
					},
				},
				Contexts: map[string]*config.Context{
					"ctx1": {
						Namespace: "default",
						ImportRef: config.ImportRef{
							LoginSourceName: "shared",
							AuthInfoName:    "utbildning-dev",
							ContextName:     "utbildning-dev",
						},
					},
				},
			},
		},
	}
}
