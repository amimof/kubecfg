package main

import (
	"fmt"
	"io"
	"os"

	"github.com/amimof/kubecfg/pkg/cmdutil"
	"github.com/amimof/kubecfg/pkg/config"
	"github.com/spf13/cobra"
)

var describeWorkspaceStdout io.Writer = os.Stdout

func newDescribeWorkspaceCmd() *cobra.Command {
	var identityFile string

	cmd := &cobra.Command{
		Use:   "workspace [WORKSPACE]",
		Short: "Show workspace details",
		Long:  `Show a workspace and its kubeconfigs in a readable format.`,
		Example: `  kubecfg describe workspace homelab
  kubecfg describe workspace homelab --identity-file ~/.config/kubecfg/age.txt`,
		Args:         cobra.MinimumNArgs(0),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runDescribeWorkspaceCmd(args, identityFile, describeWorkspaceStdout)
		}),
	}

	cmd.PersistentFlags().StringVar(&identityFile, "identity-file", "", "Age identity used to decrypt fields in configuration")

	return cmd
}

func runDescribeWorkspaceCmd(args []string, identityFile string, stdout io.Writer) error {
	compiler, err := newCompilerWithOptionalDecryptor(&cfg, identityFile)
	if err != nil {
		return err
	}

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	names := args

	if len(names) == 0 {
		for _, ws := range runtime.Workspaces {
			names = append(names, ws.Name)
		}
	}

	// Check so that all workspaces exist
	for _, name := range names {
		if !runtime.WorkspaceExists(name) {
			return fmt.Errorf("workspace does not exist: %s", name)
		}
	}

	for i, name := range names {
		if err := renderWorkspaceDescription(stdout, *runtime.Workspace(name)); err != nil {
			return fmt.Errorf("render error: %w", err)
		}
		if i != len(names)-1 {
			if err := cmdutil.RenderLine(stdout, 60); err != nil {
				return fmt.Errorf("render error: %w", err)
			}
		}
	}

	return nil
}

func renderWorkspaceDescription(stdout io.Writer, rw config.RuntimeWorkspace) error {
	containers := []*cmdutil.Container{
		cmdutil.NewContainer(nil,
			cmdutil.NewElement(`{{ "Name" | FgHiGreen }}:               {{ .Workspace.Name }}`),
			cmdutil.NewElement(`{{ "Description" | FgHiGreen }}:        {{ .Workspace.Description }}`),
			cmdutil.NewElement(`{{ "Default Kubeconfig" | FgHiGreen }}: {{ .WorkspaceDefaultKubeconfig }}`),
			cmdutil.NewElement(`{{ "Kubeconfigs" | FgHiGreen }}:        {{ .Workspace.Kubeconfigs | len | string | FgBlue }}`),
		).WithLayout(cmdutil.Layout{Dimensions: [2]int{1024, 0}}),
	}

	var i int

	for _, kubeconfig := range rw.Kubeconfigs {
		container := cmdutil.NewContainer(cmdutil.Data{
			"Kubeconfig": kubeconfig,
			"Index":      i,
		},
			cmdutil.NewElement(`{{ .Container.Index  | string | FgMagenta }}: {{ "Kubeconfig" | FgHiGreen }}:           {{ .Container.Kubeconfig.Name }}`),
			cmdutil.NewElement(`     {{ "Path" | FgHiGreen}}:                  {{ .Container.Kubeconfig.Path }}`),
			cmdutil.NewElement(`     {{ "Aliases" | FgHiGreen}}:               {{ .Container.Kubeconfig.Aliases }}`),
			cmdutil.NewElement(`     {{ "Protected" | FgHiGreen }}:            {{ .Container.Kubeconfig.Protected | string | FgYellow }}`),
			cmdutil.NewElement(`     {{ "Current Context" | FgHiGreen }}:      {{ .Container.Kubeconfig.CurrentContext.Name }}`),
			cmdutil.NewElement(`     {{ "Default Context" | FgHiGreen }}:      {{ .Container.Kubeconfig.DefaultContext.Name }}`),
			cmdutil.NewElement(`     {{ "Default Namespace" | FgHiGreen }}:    {{ .Container.Kubeconfig.DefaultNamespace }}`),
			cmdutil.NewElement(`     {{ "Default Namespace" | FgHiGreen }}:    {{ .Container.Kubeconfig.DefaultNamespace }}`),
			cmdutil.NewElement(`     {{ "Contexts" | FgHiGreen }}:             {{ .Container.Kubeconfig.Contexts | len | string | FgBlue }}`),
		).WithLayout(cmdutil.Layout{Dimensions: [2]int{1024, 0}})

		containers = append(containers, container)

		var y int

		for _, context := range kubeconfig.Contexts {
			containers = append(containers, cmdutil.NewContainer(cmdutil.Data{
				"Context": context,
				"Index":   y,
			},
				cmdutil.NewElement(`     {{ .Container.Index | string | FgMagenta}}: {{ "Context:" | FgHiGreen }}            {{ .Container.Context.Name }}`),
				cmdutil.NewElement(`        {{ "Cluster:" | FgHiGreen }}            {{ .Container.Context.Cluster.Name }}`),
				cmdutil.NewElement(`        {{ "AuthInfo:" | FgHiGreen }}           {{ .Container.Context.AuthInfo.Name }}`),
				cmdutil.NewElement(`        {{ "Namespace:"  | FgHiGreen}}          {{ .Container.Context.Namespace }}`),
				cmdutil.NewElement(`        {{ "Import:" | FgHiGreen }}`),
				cmdutil.NewElement(`          {{ "Login Source:" | FgHiGreen }}      {{ .Container.Context.Import.LoginSourceName }}`),
				cmdutil.NewElement(`          {{ "Context:" | FgHiGreen }}           {{ .Container.Context.Import.ContextName }}`),
				cmdutil.NewElement(`          {{ "Cluster:" | FgHiGreen }}           {{ .Container.Context.Import.ClusterName }}`),
				cmdutil.NewElement(`          {{ "AuthInfo:" | FgHiGreen }}          {{ .Container.Context.Import.AuthInfoName }}`),
			).WithLayout(cmdutil.Layout{Dimensions: [2]int{1024, 0}}))
			y += 1
		}
		i += 1
	}

	return cmdutil.RenderOnce(stdout, cmdutil.Data{
		"Workspace":                  rw,
		"WorkspaceDefaultKubeconfig": workspaceDefaultKubeconfigDisplay(rw.DefaultKubeconfig),
	}, containers...)
}

func workspaceDefaultKubeconfigDisplay(rk *config.RuntimeKubeconfig) string {
	if rk == nil {
		return ""
	}

	return rk.Name
}
