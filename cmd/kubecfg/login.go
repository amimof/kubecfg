package main

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/amimof/kubecfg/pkg/command"
	"github.com/amimof/kubecfg/pkg/service"
	"github.com/spf13/cobra"
)

var (
	loginStdout *bytes.Buffer = &bytes.Buffer{}
	loginStderr *bytes.Buffer = &bytes.Buffer{}
)

func newLoginCmd() *cobra.Command {
	var (
		workspaceName string
		identityFile  string
	)
	cmd := &cobra.Command{
		Use:   "login [KUBECONFIG] [CONTEXT]",
		Short: "Refresh credentials for a context",
		Long:  `Run the login flow for a kubeconfig context and write the updated credentials.`,
		Example: `  kubecfg login mainframe admin
  kubecfg login mainframe admin --workspace homelab`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runLoginCmd(workspaceName, args[0], args[1], identityFile)
		}),
	}

	cmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace")
	cmd.PersistentFlags().StringVar(&identityFile, "identity-file", "", "Age identity used to decrypt fields in configuration")

	return cmd
}

func runLoginCmd(workspaceName, kubeconfigName, contextName, identityFile string) error {
	compiler, err := newCompilerWithOptionalDecryptor(&cfg, identityFile)
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
	aui := rk.Contexts[contextName].AuthInfo

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runner := command.NewExecCommandRunner()
	loginService := service.LoginService{Runner: runner, Stdout: loginStdout, Stderr: loginStderr}
	newAuth, err := loginService.Login(ctx, aui)
	if err != nil {
		return err
	}

	rkContextRef := rk.Context(contextName)
	rk.Config.AuthInfos[rkContextRef.AuthInfo.Name] = newAuth
	rk.Config.CurrentContext = contextName

	if err := writeKubeconfig(rk.Path, *rk.Config); err != nil {
		return err
	}

	return nil
}
