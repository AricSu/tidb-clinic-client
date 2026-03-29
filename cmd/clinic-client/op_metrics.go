package main

import (
	"context"
	"io"
	"log"
	"time"

	clinicapi "github.com/AricSu/tidb-clinic-client"
	"github.com/spf13/cobra"
)

func newOPMetricsCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Run On-Premise (OP) metrics queries",
		Long:  opWorkflowHelp("Run On-Premise (OP) metrics queries for a known org and cluster."),
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "query-range",
		Short: "Query On-Premise (OP) metrics over a time range",
		Long:  opWorkflowHelp("Query On-Premise (OP) metrics over a time range."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runOPMetricsQueryRange()
		},
	})
	return cmd
}

func runOPMetricsQueryRange(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	cfg, err := loadOPConfigFromEnv(lookup, now)
	if err != nil {
		return err
	}

	client, err := clinicapi.NewClientWithConfig(clinicapi.Config{
		BaseURL:     cfg.BaseURL,
		BearerToken: cfg.APIKey,
		Timeout:     cfg.Timeout,
		Logger:      logger,
	})
	if err != nil {
		return err
	}

	result, err := client.Metrics.QueryRangeWithAutoSplit(context.Background(), clinicapi.MetricsQueryRangeRequest{
		Context: cfg.Context,
		Query:   cfg.Query,
		Start:   cfg.Start,
		End:     cfg.End,
		Step:    cfg.Step,
	})
	if err != nil {
		return err
	}

	writeMetricQueryRangeSummary(out, cfg.Query, cfg.Start, cfg.End, cfg.Step, result)
	return nil
}
