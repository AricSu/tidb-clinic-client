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

func newCloudTopSQLCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "topsql",
		Short: "Run TiDB Cloud TopSQL queries",
		Long:  cloudHelp("Run TiDB Cloud TopSQL queries for a known cluster."),
	}
	cmd.AddCommand(newCloudTopSQLSummaryCommand(deps))
	return cmd
}

func newCloudTopSQLSummaryCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "summary",
		Short: "Get TiDB Cloud TopSQL summary for a known cluster",
		Long:  cloudHelp("Get TiDB Cloud TopSQL summary for a known cluster."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudTopSQLSummary()
		},
	}
}

func runCloudTopSQLSummary(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	cfg, err := loadCloudTopSQLConfig(lookup, now)
	if err != nil {
		return err
	}
	client, err := newSDKClient(cfg.Base, logger)
	if err != nil {
		return err
	}
	target, err := resolveCloudNGMTarget(context.Background(), cfg.Base, func(ctx context.Context, req clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error) {
		return client.Cloud.GetCluster(ctx, req)
	})
	if err != nil {
		return err
	}
	result, err := client.Cloud.GetTopSQLSummary(context.Background(), clinicapi.CloudTopSQLSummaryRequest{
		Target:    target,
		Component: cfg.Component,
		Instance:  cfg.Instance,
		Start:     cfg.Start,
		End:       cfg.End,
		Top:       cfg.Top,
		Window:    cfg.Window,
		GroupBy:   cfg.GroupBy,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "total=%d\n", len(result))
	for i, row := range result {
		fmt.Fprintf(
			out,
			"topsql[%d] digest=%s cpu_time_ms=%f exec_count_per_sec=%f duration_per_exec_ms=%f query=%s\n",
			i,
			row.SQLDigest,
			row.CPUTimeMS,
			row.ExecCountPerSec,
			row.DurationPerExecMS,
			toString(row.SQLText),
		)
	}
	return nil
}
