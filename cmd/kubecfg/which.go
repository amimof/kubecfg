package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/amimof/kubecfg/pkg/cmdutil"
	"github.com/amimof/kubecfg/pkg/config"
	"github.com/spf13/cobra"
)

func newWhichCmd() *cobra.Command {
	var plain bool
	cmd := &cobra.Command{
		Use:          "which",
		Short:        "which kubeconfig is currently active",
		Long:         `Displays which kubeconfig is symlinked to ~/.kube/config`,
		Example:      `  kubecfg which`,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runWhichCmd(plain)
		}),
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "Only print file path to active kubeconfig")
	return cmd
}

func runWhichCmd(plain bool) error {
	compiler := config.NewCompiler()
	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	dst := path.Join(runtime.BaseDir, "config")
	_, err = os.Stat(dst)
	if err != nil {
		return err
	}

	sEval, err := filepath.EvalSymlinks(dst)
	if err != nil {
		return err
	}

	sBase := filepath.Base(sEval)

	if !plain {
		cmdutil.Printf(`{{ .Dst | FgGreen }} ➜ {{ .Name | FgCyan }} {{ printf "(%s)" .Symlink | FgHiBlack }}`, cmdutil.Data{"Dst": dst, "Name": sBase, "Symlink": sEval})
		return nil
	}

	fmt.Println(sEval)

	return nil
}
