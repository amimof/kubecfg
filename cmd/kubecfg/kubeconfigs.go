package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/amimof/kubecfg/pkg/config"
	"github.com/spf13/cobra"
)

var kubeconfigsStdout io.Writer = os.Stdout

func newKubeconfigsCmd() *cobra.Command {
	var workspaceName string

	cmd := &cobra.Command{
		Use:          "kubeconfigs",
		Short:        "list kubeconfigs",
		Long:         `list kubeconfigs`,
		Args:         cobra.ExactArgs(0),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runKubeconfigsCmd(workspaceName, kubeconfigsStdout)
		}),
	}

	cmd.PersistentFlags().StringVarP(&workspaceName, "workspace", "w", "", "Workspace")

	return cmd
}

func runKubeconfigsCmd(workspaceName string, stdout io.Writer) error {
	entries, err := collectKubeconfigRows(workspaceName)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "WORKSPACES\tNAME\tPATH\tALIASES\tCONTEXTS"); err != nil {
		return err
	}

	for _, entry := range entries {
		aliases := strings.Join(entry.kubeconfig.Aliases, ", ")
		contexts := 0
		if entry.kubeconfig.Contexts != nil {
			contexts = len(entry.kubeconfig.Contexts)
		}

		if _, err := fmt.Fprintf(
			tw,
			"%s\t%s\t%s\t%s\t%d\n",
			strings.Join(entry.workspaces, ", "),
			entry.name,
			entry.kubeconfig.Path,
			aliases,
			contexts,
		); err != nil {
			return err
		}
	}

	return tw.Flush()
}

type kubeconfigTableRow struct {
	name       string
	workspaces []string
	kubeconfig *config.Kubeconfig
}

func collectKubeconfigRows(workspaceName string) ([]kubeconfigTableRow, error) {
	workspaceNames := make([]string, 0, len(cfg.Workspaces))
	if workspaceName != "" {
		workspace := cfg.Workspace(workspaceName)
		if workspace == nil {
			return nil, fmt.Errorf("workspace does not exist: %s", workspaceName)
		}
		workspaceNames = append(workspaceNames, workspaceName)
	} else {
		for name := range cfg.Workspaces {
			workspaceNames = append(workspaceNames, name)
		}
		sort.Strings(workspaceNames)
	}

	rowsByName := make(map[string]*kubeconfigTableRow)
	for _, workspaceName := range workspaceNames {
		workspace := cfg.Workspace(workspaceName)
		for _, kubeconfigName := range workspace.Kubeconfigs {
			kubeconfig := cfg.Kubeconfig(kubeconfigName)
			if kubeconfig == nil {
				return nil, fmt.Errorf("workspaces.%s.kubeconfigs references missing kubeconfig %q", workspaceName, kubeconfigName)
			}

			row, ok := rowsByName[kubeconfigName]
			if !ok {
				row = &kubeconfigTableRow{name: kubeconfigName, kubeconfig: kubeconfig}
				rowsByName[kubeconfigName] = row
			}

			row.workspaces = append(row.workspaces, workspaceName)
		}
	}

	rows := make([]kubeconfigTableRow, 0, len(rowsByName))
	for _, row := range rowsByName {
		sort.Strings(row.workspaces)
		row.workspaces = compactSortedStrings(row.workspaces)
		rows = append(rows, *row)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].name < rows[j].name
	})

	return rows, nil
}

func compactSortedStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}

	result := values[:1]
	for _, value := range values[1:] {
		if value == result[len(result)-1] {
			continue
		}
		result = append(result, value)
	}

	return result
}
