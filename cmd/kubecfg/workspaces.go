package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/amimof/kubecfg/pkg/cmdutil/table"
	"github.com/amimof/kubecfg/pkg/config"
	"github.com/spf13/cobra"
)

var workspacesStdout io.Writer = os.Stdout

func newWorkspacesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspaces",
		Short:   "List workspaces",
		Long:    `List the workspaces defined in kubecfg.yaml.`,
		Aliases: []string{"ws"},
		Args:    cobra.ExactArgs(0),
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runWorkspacesCmd(workspacesStdout)
		}),
	}
	return cmd
}

func runWorkspacesCmd(stdout io.Writer) error {
	compiler := config.NewCompiler(baseDir)

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(runtime.Workspaces))
	for name := range runtime.Workspaces {
		names = append(names, name)
	}
	sort.Strings(names)

	tbl := table.NewTable([]table.Column{
		{Header: "NAME"},
		{Header: "DESCRIPTION"},
		{Header: "KUBECONFIGS"},
	})

	for _, name := range names {
		workspace := runtime.Workspace(name)
		if err := tbl.AddRow(name, workspace.Description, fmt.Sprintf("%d", len(workspace.Kubeconfigs))); err != nil {
			return err
		}
	}

	_, err = tbl.WriteTo(stdout)
	if err != nil {
		return err
	}
	return nil
}
