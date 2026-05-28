package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"

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

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tDESCRIPTION\tKUBECONFIGS"); err != nil {
		return err
	}

	for _, name := range names {
		workspace := runtime.Workspace(name)
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%d\n", name, workspace.Description, len(workspace.Kubeconfigs)); err != nil {
			return err
		}
	}

	return tw.Flush()
}
