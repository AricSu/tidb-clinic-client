package main

import "github.com/spf13/cobra"

func newCloudCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "Run TiDB Cloud Clinic queries",
	}
	cmd.AddCommand(newCloudMetricsCommand(deps))
	cmd.AddCommand(newCloudClusterDetailCommand(deps))
	cmd.AddCommand(newCloudEventsCommand(deps))
	cmd.AddCommand(newCloudTopSQLCommand(deps))
	cmd.AddCommand(newCloudSlowQueriesCommand(deps))
	return cmd
}
