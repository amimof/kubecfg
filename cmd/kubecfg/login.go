package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func newLoginCmd() *cobra.Command {
	var workspaceName string
	cmd := &cobra.Command{
		Use:   "login [KUBECONFIG] [CONTEXT]",
		Short: "Refresh credentials for a context",
		Long:  `Run the login flow for a kubeconfig context and write the updated credentials.`,
		Example: `  kubecfg login mainframe admin
  kubecfg login mainframe admin --workspace homelab`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runLoginCmd(workspaceName, args[0], args[1])
		}),
	}

	cmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace")

	return cmd
}

func runLoginCmd(workspaceName, kubeconfigName, contextName string) error {
	compiler, err := newCompilerWithOptionalDecryptor(&cfg, cfg.IdentityFiles)
	if err != nil {
		return err
	}

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	if workspaceName == "" {
		workspaceName = cfg.DefaultWorkspace
	}

	if !runtime.WorkspaceExists(workspaceName) {
		return fmt.Errorf("workspace does not exist: %s", workspaceName)
	}
	if !runtime.KubeconfigExists(workspaceName, kubeconfigName) {
		return fmt.Errorf("kubeconfig does not exist: %s/%s", workspaceName, kubeconfigName)
	}
	if !runtime.ContextExists(workspaceName, kubeconfigName, contextName) {
		return fmt.Errorf("context does not exist: %s/%s/%s", workspaceName, kubeconfigName, contextName)
	}

	// Find the credential source using workspace and kubeconfig name
	rk := runtime.Workspace(workspaceName).Kubeconfig(kubeconfigName)

	// Run login sources
	for _, source := range rk.LoginSources {
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := runLogin(ctx, source, 30*time.Second, stdout, stderr)
		cancel()
		if err != nil {
			return err
		}
	}
	if err := applyImportedContexts(rk); err != nil {
		return err
	}

	if err := writeKubeconfig(rk.Path, rk.Config); err != nil {
		return err
	}

	return nil
}
