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

func newCloudEventsCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Run TiDB Cloud event queries",
		Long:  cloudHelp("Run TiDB Cloud event queries for a known cluster."),
	}
	cmd.AddCommand(newCloudEventsQueryCommand(deps))
	cmd.AddCommand(newCloudEventsDetailCommand(deps))
	return cmd
}

func newCloudEventsQueryCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "query",
		Short: "Query TiDB Cloud events for a known cluster and time window",
		Long:  cloudHelp("Query TiDB Cloud events for a known cluster and time window."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudEventsQuery()
		},
	}
}

func newCloudEventsDetailCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "detail",
		Short: "Get TiDB Cloud event detail for a known cluster and event ID",
		Long:  cloudHelp("Get TiDB Cloud event detail for a known cluster and event ID."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudEventsDetail()
		},
	}
}

func runCloudEventsQuery(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	cfg, err := loadConfigFromEnv(lookup, now)
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

	target, err := resolveCloudTarget(context.Background(), cfg, func(ctx context.Context, req clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error) {
		return client.Cloud.GetCluster(ctx, req)
	})
	if err != nil {
		return err
	}

	result, err := client.Cloud.QueryEvents(context.Background(), clinicapi.CloudEventsRequest{
		Target:    target,
		StartTime: cfg.Start,
		EndTime:   cfg.End,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "total=%d\n", result.Total)
	for i, event := range result.Events {
		fmt.Fprintf(
			out,
			"event[%d] id=%s name=%s display_name=%s create_time=%d\n",
			i,
			event.EventID,
			event.Name,
			event.DisplayName,
			event.CreateTime,
		)
	}
	return nil
}

func runCloudEventsDetail(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	cfg, err := loadCloudEventDetailConfig(lookup, now)
	if err != nil {
		return err
	}

	client, err := newSDKClient(cfg.Base, logger)
	if err != nil {
		return err
	}

	target, err := resolveCloudTarget(context.Background(), cfg.Base, func(ctx context.Context, req clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error) {
		return client.Cloud.GetCluster(ctx, req)
	})
	if err != nil {
		return err
	}

	result, err := client.Cloud.GetEventDetail(context.Background(), clinicapi.CloudEventDetailRequest{
		Target:  target,
		EventID: cfg.EventID,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "event_id=%s\n", cfg.EventID)
	fmt.Fprintf(out, "detail=%v\n", result)
	return nil
}
