package main

import "github.com/spf13/cobra"

func newOPCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "op",
		Short: "Run On-Premise (OP) Clinic queries",
	}
	cmd.AddCommand(newOPMetricsCommand(deps))
	cmd.AddCommand(newOPLogsCommand(deps))
	cmd.AddCommand(newOPSlowQueriesCommand(deps))
	cmd.AddCommand(newOPConfigsCommand(deps))
	return cmd
}
