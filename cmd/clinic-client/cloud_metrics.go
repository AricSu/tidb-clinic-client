package main

import "github.com/spf13/cobra"

func newCloudMetricsCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Run TiDB Cloud metrics queries",
		Long:  sharedMetricsHelp(),
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "query-range",
		Short: "Query TiDB Cloud metrics over a time range",
		Long:  sharedMetricsHelp(),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudMetricsQueryRange()
		},
	})
	return cmd
}
