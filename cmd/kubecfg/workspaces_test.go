package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRunWorkspacesCmdPrintsHeaderedTable(t *testing.T) {
	originalCfg := cfg
	originalBaseDir := baseDir
	t.Cleanup(func() {
		cfg = originalCfg
		baseDir = originalBaseDir
	})

	cfg = newWorkspacesCommandTestConfig()
	baseDir = "/tmp"

	var stdout bytes.Buffer
	err := runWorkspacesCmd(&stdout)
	require.NoError(t, err)
	require.Equal(t, []string{
		"NAME       DESCRIPTION         KUBECONFIGS",
		"default    Main workspace      2",
		"secondary  Ops clusters        1",
		"third      Untitled workspace  1",
	}, strings.Split(strings.TrimSpace(stdout.String()), "\n"))
}

func newWorkspacesCommandTestConfig() config.Config {
	return config.Config{
		Version: "v1",
		Workspaces: map[string]*config.Workspace{
			"default": {
				Description: "Main workspace",
				Kubeconfigs: []string{"beta", "alpha"},
			},
			"secondary": {
				Description: "Ops clusters",
				Kubeconfigs: []string{"gamma"},
			},
			"third": {
				Description: "Untitled workspace",
				Kubeconfigs: []string{"alpha"},
			},
		},
		Kubeconfigs: map[string]*config.Kubeconfig{
			"alpha": {
				Path: "/tmp/a.yaml",
				Clusters: map[string]*config.Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*config.AuthInfo{
					"user": {},
				},
				Contexts: map[string]*config.Context{
					"admin": {Cluster: "cluster", AuthInfo: "user"},
				},
			},
			"beta": {
				Path: "/tmp/b.yaml",
				Clusters: map[string]*config.Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*config.AuthInfo{
					"user": {},
				},
				Contexts: map[string]*config.Context{
					"admin": {Cluster: "cluster", AuthInfo: "user"},
				},
			},
			"gamma": {
				Path: "/tmp/c.yaml",
				Clusters: map[string]*config.Cluster{
					"cluster": {Server: "https://example.com"},
				},
				AuthInfos: map[string]*config.AuthInfo{
					"user": {},
				},
				Contexts: map[string]*config.Context{
					"admin": {Cluster: "cluster", AuthInfo: "user"},
				},
			},
		},
	}
}
