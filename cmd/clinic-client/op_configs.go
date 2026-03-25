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

func newOPConfigsCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "configs",
		Short: "Fetch On-Premise (OP) config snapshots with automatic item selection",
		Long:  opWorkflowHelp("Fetch On-Premise (OP) config snapshots with automatic item selection."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runOPConfigsGet()
		},
	}
}

func runOPConfigsGet(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	cfg, err := loadOPItemRequestConfig(lookup, now)
	if err != nil {
		return err
	}
	client, err := newSDKClient(cfg.Base, logger)
	if err != nil {
		return err
	}
	item, err := resolveOPCatalogItem(context.Background(), client, cfg.Base, opCatalogIntentConfigs)
	if err != nil {
		return err
	}
	result, err := client.Configs.Get(context.Background(), clinicapi.ConfigRequest{
		Context: cfg.Base.Context,
		ItemID:  item.ItemID,
	})
	if err != nil {
		return err
	}
	writeResolvedCatalogItem(out, item)
	fmt.Fprintf(out, "total=%d\n", len(result.Entries))
	for i, entry := range result.Entries {
		fmt.Fprintf(
			out,
			"config[%d] component=%s key=%s value=%s source_ref=%s\n",
			i,
			entry.Component,
			entry.Key,
			toString(entry.Value),
			entry.SourceRef,
		)
	}
	return nil
}
