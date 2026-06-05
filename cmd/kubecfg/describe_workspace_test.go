package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRunDescribeWorkspaceCmdRendersDefaultKubeconfig(t *testing.T) {
	originalCfg := cfg
	t.Cleanup(func() {
		cfg = originalCfg
	})

	cfg = newDescribeWorkspaceTestConfig()

	var stdout bytes.Buffer
	err := runDescribeWorkspaceCmd([]string{"work"}, &stdout)
	require.NoError(t, err)

	output := stdout.String()
	require.Contains(t, output, "Default Kubeconfig")
	require.Contains(t, output, "vgr")
	require.NotContains(t, output, "<nil>")
}

func TestWorkspaceDefaultKubeconfigDisplay(t *testing.T) {
	require.Equal(t, "", workspaceDefaultKubeconfigDisplay(nil))
	require.Equal(t, "vgr", workspaceDefaultKubeconfigDisplay(&config.RuntimeKubeconfig{Name: "vgr"}))
}

func newDescribeWorkspaceTestConfig() config.Config {
	return config.Config{
		Version: "v1",
		Workspaces: map[string]*config.Workspace{
			"work": {
				Kubeconfigs:       []string{"vgr"},
				DefaultKubeconfig: "vgr",
			},
		},
		Kubeconfigs: map[string]*config.Kubeconfig{
			"vgr": {
				Path:           "/tmp/vgr.yaml",
				DefaultContext: "ops",
				Clusters: map[string]*config.Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*config.AuthInfo{
					"user": {},
				},
				Contexts: map[string]*config.Context{
					"admin": {Cluster: "cluster", AuthInfo: "user"},
					"ops":   {Cluster: "cluster", AuthInfo: "user"},
				},
			},
		},
	}
}

func TestRunDescribeWorkspaceCmdRendersEmptyDefaultKubeconfigWhenUnset(t *testing.T) {
	originalCfg := cfg
	t.Cleanup(func() {
		cfg = originalCfg
	})

	cfg = newDescribeWorkspaceTestConfig()
	cfg.Workspaces["work"].DefaultKubeconfig = ""

	var stdout bytes.Buffer
	err := runDescribeWorkspaceCmd([]string{"work"}, &stdout)
	require.NoError(t, err)

	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.Contains(line, "Default Kubeconfig") {
			require.NotContains(t, line, "<nil>")
			return
		}
	}

	t.Fatal("default kubeconfig line not found")
}
