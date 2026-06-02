package main

import (
	"os"
	"path"
	"path/filepath"

	"github.com/amimof/kubecfg/pkg/cmdutil"
	"github.com/amimof/kubecfg/pkg/config"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

func newUseCmd() *cobra.Command {
	var glob []string
	cmd := &cobra.Command{
		Use:          "use",
		Short:        "Use a rendered kubeconfig",
		Long:         `Select and activate an existing kubeconfig file`,
		Example:      `  kubecfg use`,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runUseCmd(glob)
		}),
	}
	h, _ := os.UserHomeDir()

	cmd.Flags().StringArrayVar(&glob, "glob", []string{path.Join(h, ".kube/*.yaml")}, "List files matching a pattern to include. This flag can be used multiple times.")

	return cmd
}

func runUseCmd(glob []string) error {
	selected, err := pickKubeconfig(glob)
	if err != nil {
		return err
	}

	compiler := config.NewCompiler()
	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	err = setConfig(runtime.BaseDir, selected)
	if err != nil {
		return err
	}

	cmdutil.Printf(`{{ "✔" | FgGreen }} Using kubeconfig {{ .Kubeconfig | FgCyan }}`, cmdutil.Data{"Kubeconfig": selected})
	return nil
}

func pickKubeconfig(globs []string) (string, error) {
	// Assemble items in Cfg
	kubeconfigs := ListKubeconfigs(globs)

	inputChan := make(chan string)
	go func() {
		for _, name := range kubeconfigs {
			inputChan <- name
		}
		close(inputChan)
	}()

	selected, err := pick(inputChan)
	if err != nil {
		return "", err
	}

	return selected, nil
}

func ListKubeconfigs(globs []string) []string {
	kubeConfigs := []string{}
	for _, iglobs := range globs {
		matches, err := filepath.Glob(iglobs)
		if err != nil {
			continue
		}
		kubeConfigs = append(kubeConfigs, matches...)
	}

	var res []string

	for _, kubeConfig := range kubeConfigs {
		if info, err := os.Stat(kubeConfig); err == nil {
			if !info.IsDir() {

				_, err := clientcmd.LoadFromFile(kubeConfig)
				if err != nil {
					return nil
				}

				res = append(res, info.Name())
			}
		}
	}
	return res
}
