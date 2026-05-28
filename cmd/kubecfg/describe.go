package main

import (
	"github.com/spf13/cobra"
)

func newDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "describe [RESOURCE]",
		Short:        "Show resource details",
		Long:         `Show details for a supported resource.`,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
	}

	cmd.AddCommand(newDescribeWorkspaceCmd())

	return cmd
}
