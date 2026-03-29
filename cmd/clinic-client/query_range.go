package main

import (
	"context"
	"io"
	"log"
	"time"

	clinicapi "github.com/AricSu/tidb-clinic-client"
)

func runMetricsQueryRange(
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
	reqCtx, err := resolveRequestContext(context.Background(), cfg, func(ctx context.Context, req clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error) {
		return client.Cloud.GetCluster(ctx, req)
	})
	if err != nil {
		return err
	}

	result, err := client.Metrics.QueryRangeWithAutoSplit(context.Background(), clinicapi.MetricsQueryRangeRequest{
		Context: reqCtx,
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
