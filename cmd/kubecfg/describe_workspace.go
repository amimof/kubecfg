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
		Use:          "workspace [WORKSPACE]",
		Short:        "describe workspace",
		Long:         `describe workspace`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: withConfig(func(cmd *cobra.Command, args []string) error {
			return runDescribeWorkspaceCmd(args[0], identityFile, describeWorkspaceStdout)
		}),
	}

	cmd.PersistentFlags().StringVar(&identityFile, "identity-file", "", "Age identity used to decrypt fields in configuration")

	return cmd
}

func runDescribeWorkspaceCmd(name, identityFile string, stdout io.Writer) error {
	compiler, err := newCompilerWithOptionalDecryptor(&cfg, identityFile)
	if err != nil {
		return err
	}

	runtime, err := compiler.Compile(&cfg)
	if err != nil {
		return err
	}

	if !runtime.WorkspaceExists(name) {
		return fmt.Errorf("workspace does not exist: %s", name)
	}

	return renderWorkspaceDescription(stdout, *runtime.Workspace(name))
}

func renderWorkspaceDescription(stdout io.Writer, rw config.RuntimeWorkspace) error {
	containers := []*cmdutil.Container{
		cmdutil.NewContainer(nil,
			cmdutil.NewElement(`{{ "Name" | FgHiGreen }}:               {{ .Workspace.Name }}`),
			cmdutil.NewElement(`{{ "Description" | FgHiGreen }}:        {{ .Workspace.Description }}`),
			cmdutil.NewElement(`{{ "Default Namespace" | FgHiGreen }}:  {{ .Workspace.DefaultNamespace }}`),
			cmdutil.NewElement(`{{ "Kubeconfigs" | FgHiGreen }}:        {{ .Workspace.Kubeconfigs | len | string | FgHiBlue }}`),
		).WithLayout(cmdutil.Layout{Dimensions: [2]int{1024, 0}}),
	}

	for _, kubeconfig := range rw.Kubeconfigs {
		container := cmdutil.NewContainer(cmdutil.Data{
			"Kubeconfig": kubeconfig,
		},
			cmdutil.NewElement(`{{ "Kubeconfig" | FgHiGreen }}:         {{ .Container.Kubeconfig.Name }}`),
			cmdutil.NewElement(`{{ "Path" | FgHiGreen}}:               {{ .Container.Kubeconfig.Path }}`),
			cmdutil.NewElement(`{{ "Aliases" | FgHiGreen}}:            {{ .Container.Kubeconfig.Aliases }}`),
			cmdutil.NewElement(`{{ "Protected" | FgHiGreen }}:          {{ .Container.Kubeconfig.Protected | string | FgHiYellow }}`),
			cmdutil.NewElement(`{{ "Current Context" | FgHiGreen }}:    {{ .Container.Kubeconfig.CurrentContext }}`),
			cmdutil.NewElement(`{{ "Contexts" | FgHiGreen }}:           {{ .Container.Kubeconfig.Contexts | len | string | FgHiBlue }}`),
		).WithLayout(cmdutil.Layout{Dimensions: [2]int{1024, 0}})

		containers = append(containers, container)

		for _, context := range kubeconfig.Contexts {
			containers = append(containers, cmdutil.NewContainer(cmdutil.Data{
				"Context": context,
			},
				cmdutil.NewElement(`  {{ "Context:" | FgHiGreen }}            {{ .Container.Context.Name }}`),
				cmdutil.NewElement(`  {{ "Cluster:" | FgHiGreen }}            {{ .Container.Context.Cluster.Name }}`),
				cmdutil.NewElement(`  {{ "AuthInfo:" | FgHiGreen }}           {{ .Container.Context.AuthInfo.Name }}`),
				cmdutil.NewElement(`  {{ "Namespace:"  | FgHiGreen}}          {{ .Container.Context.Namespace }}`),
				cmdutil.NewElement(`  {{ "Credential Source:"  | FgHiGreen}}  {{ .Container.Context.AuthInfo.CredentialSource | source }}`),
			).WithLayout(cmdutil.Layout{Dimensions: [2]int{1024, 0}}))
		}
	}

	return cmdutil.RenderOnce(stdout, cmdutil.Data{"Workspace": rw}, containers...)
}
