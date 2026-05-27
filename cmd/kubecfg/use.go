package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/amimof/kubecfg/pkg/config"
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
	var workspaceName string

	cmd := &cobra.Command{
		Use:          "use [NAME]",
		Short:        "use workspace",
		Long:         `use workspace`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runUseCmdFzf(workspaceName)
			}
			return runUseCmd(workspaceName, args[0])
		}),
	}

	cmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace")

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

func runUseCmd(workspaceName, kubeconfigName string) error {
	compiler := config.NewCompiler(baseDir)

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
	if err := writeKubeconfig(rk.Path, *rk.Config); err != nil {
		return err
	}

	if err := setConfig(rk.Path); err != nil {
		return err
	}

	fmt.Printf("Using kubeconfig: %s/%s\n", workspaceName, kubeconfigName)
	return nil
}

func runUseCmdFzf(workspaceName string) error {
	compiler := config.NewCompiler(baseDir)

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
	if err := writeKubeconfig(rk.Path, *rk.Config); err != nil {
		return err
	}
	if err := setConfig(rk.Path); err != nil {
		return err
	}
	fmt.Printf("Using kubeconfig: %s/%s\n", workspace, selected)
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
