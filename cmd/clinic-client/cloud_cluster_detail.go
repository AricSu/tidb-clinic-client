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

func newCloudClusterDetailCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "cluster-detail",
		Short: "Get TiDB Cloud cluster detail for a known cluster ID",
		Long:  cloudHelp("Get TiDB Cloud cluster detail for a known cluster ID."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runCloudClusterDetail()
		},
	}
}

func runCloudClusterDetail(
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

	cloudCluster, err := resolveCloudCluster(context.Background(), cfg, func(ctx context.Context, req clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error) {
		return client.Cloud.GetCluster(ctx, req)
	})
	if err != nil {
		return err
	}

	detail, err := client.Cloud.GetClusterDetail(context.Background(), clinicapi.CloudClusterDetailRequest{
		Target: cloudCluster.CloudTarget(),
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "cluster_id=%s\n", cloudCluster.ClusterID)
	fmt.Fprintf(out, "detail_id=%s\n", detail.ID)
	fmt.Fprintf(out, "name=%s\n", detail.Name)
	fmt.Fprintf(out, "topology=%s\n", detail.Topology())
	return nil
}
