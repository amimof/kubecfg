package main

import (
	"fmt"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/spf13/cobra"
)

func newWorkspacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspaces",
		Short:   "list workspaces",
		Long:    `list workspaces`,
		Aliases: []string{"ws"},
		Args:    cobra.ExactArgs(0),
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runWorkspacesCmd()
		}),
	}
	return cmd
}

func runWorkspacesCmd() error {
	compiler := config.NewCompiler(baseDir)

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	for name, k := range runtime.Workspaces {
		fmt.Println(name, k.Description, len(k.Kubeconfigs))
	}

	return nil
}
