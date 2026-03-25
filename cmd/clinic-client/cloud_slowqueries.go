package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	clinicapi "github.com/aric/tidb-clinic-client"
	"github.com/spf13/cobra"
)

func newCloudSlowQueriesCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slowqueries",
		Short: "Inspect TiDB Cloud slow queries",
		Long:  cloudHelp("Inspect TiDB Cloud slow queries for a known cluster."),
	}
	cmd.AddCommand(newCloudTopSlowQueriesCommand(deps))
	cmd.AddCommand(newCloudSlowQueriesListCommand(deps))
	cmd.AddCommand(newCloudSlowQueriesDetailCommand(deps))
	return cmd
}

func newCloudTopSlowQueriesCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "top",
		Short: "Get TiDB Cloud top slow query summary",
		Long:  cloudHelp("Get TiDB Cloud top slow query summary."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudTopSlowQueries()
		},
	}
}

func newCloudSlowQueriesListCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List TiDB Cloud slow query samples for a digest",
		Long:  cloudHelp("List TiDB Cloud slow query samples for a digest."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudSlowQueriesList()
		},
	}
}

func newCloudSlowQueriesDetailCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "detail",
		Short: "Get TiDB Cloud slow query detail for a single sample",
		Long:  cloudHelp("Get TiDB Cloud slow query detail for a single sample."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudSlowQueriesDetail()
		},
	}
}

func runCloudTopSlowQueries(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	cfg, err := loadCloudTopSlowQueriesConfig(lookup, now)
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
	result, err := client.Cloud.GetTopSlowQueries(context.Background(), clinicapi.CloudTopSlowQueriesRequest{
		Target:  target,
		Start:   cfg.Start,
		Hours:   cfg.Hours,
		OrderBy: cfg.OrderBy,
		Limit:   cfg.Limit,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "total=%d\n", len(result))
	for i, row := range result {
		fmt.Fprintf(
			out,
			"slowquery_top[%d] digest=%s db=%s count=%d sum_latency=%f max_latency=%f avg_latency=%f query=%s\n",
			i,
			row.SQLDigest,
			row.DB,
			row.Count,
			row.SumLatency,
			row.MaxLatency,
			row.AvgLatency,
			toString(row.SQLText),
		)
	}
	return nil
}

func runCloudSlowQueriesList(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	cfg, err := loadCloudSlowQueryListConfig(lookup, now)
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
	result, err := client.Cloud.ListSlowQueries(context.Background(), clinicapi.CloudSlowQueryListRequest{
		Target:  target,
		Digest:  cfg.Digest,
		Start:   cfg.Start,
		End:     cfg.End,
		OrderBy: cfg.OrderBy,
		Limit:   cfg.Limit,
		Desc:    cfg.Desc,
		Fields:  cfg.Fields,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "total=%d\n", len(result))
	for i, row := range result {
		fmt.Fprintf(
			out,
			"slowquery_list[%d] digest=%s timestamp=%s query_time=%f connection_id=%s query=%s\n",
			i,
			row.Digest,
			row.Timestamp,
			row.QueryTime,
			row.ConnectionID,
			toString(row.Query),
		)
	}
	return nil
}

func runCloudSlowQueriesDetail(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	cfg, err := loadCloudSlowQueryDetailConfig(lookup, now)
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
	result, err := client.Cloud.GetSlowQueryDetail(context.Background(), clinicapi.CloudSlowQueryDetailRequest{
		Target:       target,
		Digest:       cfg.Digest,
		ConnectionID: cfg.ConnectionID,
		Timestamp:    cfg.Timestamp,
	})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "digest=%s\n", cfg.Digest)
	fmt.Fprintf(out, "connection_id=%s\n", cfg.ConnectionID)
	fmt.Fprintf(out, "timestamp=%s\n", cfg.Timestamp)
	fmt.Fprintf(out, "detail=%v\n", result)
	return nil
}
