package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/amimof/kubecfg/pkg/cmdutil"
	"github.com/amimof/kubecfg/pkg/command"
	"github.com/amimof/kubecfg/pkg/config"
	"github.com/amimof/kubecfg/pkg/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	fzf "github.com/junegunn/fzf/src"
)

var (
	errNoSelection = errors.New("no selection")
	fzfRun         = fzf.Run
)

func newUseCmd() *cobra.Command {
	var (
		workspaceName string
		skipLogin     bool
		identityFile  string
		waitTimeout   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "use [NAME]",
		Short: "Select and write a kubeconfig",
		Long:  `Select a kubeconfig and write it to the base directory.`,
		Example: `  kubecfg use
  kubecfg use homelab/mainframe`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runUseCmdFzf(workspaceName, skipLogin, identityFile, waitTimeout)
			}
			return runUseCmd(workspaceName, args[0], skipLogin, identityFile, waitTimeout)
		}),
	}

	cmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace")
	cmd.PersistentFlags().StringVar(&identityFile, "identity-file", "", "Age identity used to decrypt fields in configuration")
	cmd.PersistentFlags().BoolVar(&skipLogin, "skip-login", false, "Skip execution of login flow prior to kubeconfig rendering")
	cmd.PersistentFlags().DurationVar(&waitTimeout, "timeout", time.Second*30, "How long in seconds to wait for login opearation to finish before giving up")

	return cmd
}

func writeKubeconfig(path string, kubeconfig api.Config) error {
	data, err := clientcmd.Write(kubeconfig)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// setConfig creates a new symlink to a kubeconfigfile overwriting any existing one
func setConfig(name string) error {
	dst := path.Join(baseDir, "config")
	// Remove existing symlink to config so we don't run into an error
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		err := os.Remove(dst)
		if err != nil {
			return err
		}
	}
	// Create the symlink to config
	return os.Symlink(name, path.Join(baseDir, "config"))
}

func runUseCmd(workspaceName, kubeconfigName string, skipLogin bool, identityFile string, waitTimeout time.Duration) error {
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

	if strings.Contains(kubeconfigName, "/") {
		ss := strings.Split(kubeconfigName, "/")
		if len(ss) == 2 {
			workspaceName = ss[0]
			kubeconfigName = ss[1]
		}
	}

	if kubeconfigName != "" {
		if !runtime.WorkspaceExists(workspaceName) {
			return fmt.Errorf("workspace does not exist: %s", kubeconfigName)
		}
		if !runtime.KubeconfigExists(workspaceName, kubeconfigName) {
			return fmt.Errorf("kubeconfig does not exist: %s/%s", workspaceName, kubeconfigName)
		}
	}

	rk := runtime.Workspace(workspaceName).Kubeconfig(kubeconfigName)

	if rk.Config.CurrentContext == "" {
		rk.Config.CurrentContext = rk.Name
	}

	if !skipLogin {
		if err := runLogin(rk, waitTimeout); err != nil {
			return err
		}
	}

	if err := writeKubeconfig(rk.Path, *rk.Config); err != nil {
		return err
	}

	if err := setConfig(rk.Path); err != nil {
		return err
	}

	fmt.Printf("Using kubeconfig: %s/%s\n", workspaceName, kubeconfigName)
	return nil
}

func runUseCmdFzf(workspaceName string, skipLogin bool, identityFile string, waitTimeout time.Duration) error {
	compiler, err := newCompilerWithOptionalDecryptor(&cfg, identityFile)
	if err != nil {
		return err
	}

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	workspace, selected, err := pickContext(runtime, workspaceName)
	if err != nil {
		if errors.Is(err, errNoSelection) {
			return nil
		}
		return err
	}

	rk := runtime.Workspace(workspace).Kubeconfig(selected)

	if rk.Config.CurrentContext == "" {
		rk.Config.CurrentContext = rk.Name
	}

	if !skipLogin {
		if err := runLogin(rk, waitTimeout); err != nil {
			return err
		}
	}

	if err := writeKubeconfig(rk.Path, *rk.Config); err != nil {
		return err
	}
	if err := setConfig(rk.Path); err != nil {
		return err
	}
	fmt.Printf("Using kubeconfig: %s/%s\n", workspace, selected)
	return nil
}

func runLogin(rk *config.RuntimeKubeconfig, waitTimeout time.Duration) error {
	for name, ctx := range rk.Contexts {

		aui := rk.AuthInfo(ctx.AuthInfo.Name)

		if aui.CredentialSource != nil {
			cmdCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			dash, err := cmdutil.NewDashboard([]string{name}, cmdutil.WithHeader("Running login flow for"))
			if err != nil {
				logrus.Fatal(err)
			}

			go dash.Loop(cmdCtx)

			// Fire off start operations concurrently
			go func() {
				dash.FailAfterMsg(0, waitTimeout, "timeout reached")

				runner := command.NewExecCommandRunner()
				loginService := service.LoginService{Runner: runner, Stdout: loginStdout, Stderr: loginStderr}

				newAuth, err := loginService.Login(cmdCtx, aui)
				if err != nil {
					escapedMsg := strings.ReplaceAll(loginStdout.String(), "\n", "n")
					dash.FailMsg(0, "Login command returned an error")
					dash.SetPhase(0, escapedMsg)
					logrus.Fatal(err)
				}

				rk.Config.AuthInfos[ctx.AuthInfo.Name] = newAuth

				dash.DoneMsg(0, fmt.Sprintf("Successfully logged in user %s", ctx.AuthInfo.Name))
			}()

			dash.WaitAnd(cancel)

		}
	}

	return nil
}

func pickContext(rc *config.RuntimeConfig, workspaceName string) (string, string, error) {
	if workspaceName != "" {
		if _, ok := rc.Workspaces[workspaceName]; !ok {
			return "", "", fmt.Errorf("workspace does not exist: %s", workspaceName)
		}
	}

	inputChan := make(chan string)
	go func() {
		for name, w := range rc.Workspaces {
			for _, k := range w.Kubeconfigs {
				input := fmt.Sprintf("%s/%s", w.Name, k.Name)

				if workspaceName != "" && workspaceName != name {
					continue
				}
				inputChan <- input

			}
		}
		close(inputChan)
	}()

	outputChan := make(chan string, 1)

	// Build fzf.Options
	options, err := fzf.ParseOptions(
		true,
		[]string{"--reverse", "--border", "--height=40%"},
	)
	if err != nil {
		return "", "", fmt.Errorf("fzf exit error %d: %w", fzf.ExitError, err)
	}

	// Set up input and output channels
	options.Input = inputChan
	options.Output = outputChan

	// Run fzf
	code, err := fzfRun(options)
	if err != nil {
		return "", "", fmt.Errorf("fzf exited with code %d: %w", code, err)
	}

	switch code {
	case fzf.ExitInterrupt, fzf.ExitNoMatch:
		return "", "", errNoSelection
	case fzf.ExitOk:
	default:
		return "", "", fmt.Errorf("fzf exited with code %d", code)
	}

	var selected string
	select {
	case selected = <-outputChan:
	default:
		return "", "", fmt.Errorf("fzf exited with code %d without a selection", code)
	}

	ss := strings.Split(selected, "/")
	if len(ss) == 2 {
		return ss[0], ss[1], nil
	}

	return workspaceName, selected, nil
}
