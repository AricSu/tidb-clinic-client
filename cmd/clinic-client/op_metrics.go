package main

import (
	"context"
	"fmt"
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

	fmt.Fprintf(out, "query=%s\n", cfg.Query)
	fmt.Fprintf(out, "window=%d..%d step=%s\n", cfg.Start, cfg.End, cfg.Step)
	fmt.Fprintf(out, "result_type=%s partial=%t series=%d\n", result.ResultType, result.IsPartial, len(result.Series))
	if len(result.Series) > 0 {
		fmt.Fprintf(out, "first_series_labels=%v samples=%d\n", result.Series[0].Labels, len(result.Series[0].Values))
	}
	return nil
}
