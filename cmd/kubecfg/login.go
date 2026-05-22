package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/spf13/cobra"
)

var (
	loginStdout io.Writer = os.Stdout
	loginStderr io.Writer = os.Stderr
)

func newLoginCmd() *cobra.Command {
	var workspaceName string
	cmd := &cobra.Command{
		Use:          "login [KUBECONFIG] [CONTEXT]",
		Short:        "login to cluster in provided cluster",
		Long:         `login to cluster in provided cluster and generate kubeconfig with credentials`,
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
	compiler := config.NewCompiler(baseDir)

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
	credSource := rk.Contexts[contextName].AuthInfo.CredentialSource

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create temporary file where kubeconfig is written to by the exec command
	tmpFile, err := os.CreateTemp("/tmp", "kubecfg-login")
	if err != nil {
		return err
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			panic(err)
		}
		if err := os.Remove(tmpFile.Name()); err != nil {
			panic(err)
		}
	}()

	// Create command
	c := exec.CommandContext(ctx, credSource.Command, credSource.Args...)
	c.Env = credSource.Env
	c.Env = append(c.Env, fmt.Sprintf("%s=%s", "KUBECONFIG", path.Join(tmpFile.Name())))
	c.Stdout = loginStdout
	c.Stderr = loginStderr

	// Run command and stream output to avoid pipe deadlocks.
	if err := c.Run(); err != nil {
		return err
	}

	// Read temporary kubeconfig to extract token
	kubeconfig, err := clientcmd.LoadFromFile(tmpFile.Name())
	if err != nil {
		return err
	}

	if _, ok := kubeconfig.Contexts[credSource.Import.Context]; !ok {
		return fmt.Errorf("could not import auth info from context %s", credSource.Import.Context)
	}

	authInfoRef := kubeconfig.Contexts[credSource.Import.Context].AuthInfo
	authInfo := kubeconfig.AuthInfos[authInfoRef]

	rkContextRef := rk.Context(contextName)
	rk.Config.AuthInfos[rkContextRef.Name] = authInfo

	if err := writeKubeconfig(rk.Path, *rk.Config); err != nil {
		return err
	}

	return nil
}
