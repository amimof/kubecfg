package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRunKubeconfigsCmdUsesExplicitWorkspace(t *testing.T) {
	originalCfg := cfg
	t.Cleanup(func() {
		cfg = originalCfg
	})

	cfg = newKubeconfigsCommandTestConfig()

	var stdout bytes.Buffer
	err := runKubeconfigsCmd("secondary", &stdout)
	require.NoError(t, err)
	require.Equal(t, []string{
		"NAME   WORKSPACES  PATH         ALIASES  CONTEXTS",
		"alpha  secondary   /tmp/a.yaml  a1, a2   1",
		"beta   secondary   /tmp/b.yaml           2",
	}, strings.Split(strings.TrimSpace(stdout.String()), "\n"))
}

func TestRunKubeconfigsCmdListsAllWorkspacesWhenFilterIsOmitted(t *testing.T) {
	originalCfg := cfg
	t.Cleanup(func() {
		cfg = originalCfg
	})

	cfg = newKubeconfigsCommandTestConfig()

	var stdout bytes.Buffer
	err := runKubeconfigsCmd("", &stdout)
	require.NoError(t, err)
	require.Equal(t, []string{
		"NAME   WORKSPACES          PATH         ALIASES  CONTEXTS",
		"alpha  secondary           /tmp/a.yaml  a1, a2   1",
		"beta   default, secondary  /tmp/b.yaml           2",
		"gamma  default             /tmp/c.yaml  g        0",
	}, strings.Split(strings.TrimSpace(stdout.String()), "\n"))
}

func TestRunKubeconfigsCmdReturnsErrorForUnknownWorkspace(t *testing.T) {
	originalCfg := cfg
	t.Cleanup(func() {
		cfg = originalCfg
	})

	cfg = newKubeconfigsCommandTestConfig()

	var stdout bytes.Buffer
	err := runKubeconfigsCmd("missing", &stdout)
	require.EqualError(t, err, "workspace does not exist: missing")
}

func TestRunKubeconfigsCmdReturnsErrorForMissingReferencedKubeconfig(t *testing.T) {
	originalCfg := cfg
	t.Cleanup(func() {
		cfg = originalCfg
	})

	cfg = newKubeconfigsCommandTestConfig()
	cfg.Workspaces["default"].Kubeconfigs = append(cfg.Workspaces["default"].Kubeconfigs, "missing")

	var stdout bytes.Buffer
	err := runKubeconfigsCmd("default", &stdout)
	require.EqualError(t, err, "workspaces.default.kubeconfigs references missing kubeconfig \"missing\"")
}

func newKubeconfigsCommandTestConfig() config.Config {
	return config.Config{
		Version:          "v1",
		DefaultWorkspace: "default",
		Workspaces: map[string]*config.Workspace{
			"default": {
				Kubeconfigs: []string{"gamma", "beta"},
			},
			"secondary": {
				Kubeconfigs: []string{"beta", "alpha"},
			},
		},
		Kubeconfigs: map[string]*config.Kubeconfig{
			"alpha": {
				Path:    "/tmp/a.yaml",
				Aliases: []string{"a1", "a2"},
				Contexts: map[string]*config.Context{
					"admin": {},
				},
			},
			"beta": {
				Path: "/tmp/b.yaml",
				Contexts: map[string]*config.Context{
					"admin": {},
					"ops":   {},
				},
			},
			"gamma": {
				Path:    "/tmp/c.yaml",
				Aliases: []string{"g"},
			},
		},
	}
}
