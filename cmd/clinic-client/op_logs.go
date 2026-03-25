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

func newOPLogsCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Search On-Premise (OP) logs with automatic item selection",
		Long:  opWorkflowHelp("Search On-Premise (OP) logs with automatic item selection."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runOPLogsSearch()
		},
	}
}

func runOPLogsSearch(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	cfg, err := loadOPItemRequestConfig(lookup, now)
	if err != nil {
		return err
	}
	client, err := newSDKClient(cfg.Base, logger)
	if err != nil {
		return err
	}
	item, err := resolveOPCatalogItem(context.Background(), client, cfg.Base, opCatalogIntentLogs)
	if err != nil {
		return err
	}
	result, err := client.Logs.Search(context.Background(), clinicapi.LogSearchRequest{
		Context:   cfg.Base.Context,
		ItemID:    item.ItemID,
		StartTime: cfg.Base.Start,
		EndTime:   cfg.Base.End,
		Pattern:   cfg.Pattern,
		Limit:     cfg.Limit,
	})
	if err != nil {
		return err
	}
	writeResolvedCatalogItem(out, item)
	fmt.Fprintf(out, "total=%d\n", result.Total)
	for i, record := range result.Records {
		fmt.Fprintf(
			out,
			"log[%d] timestamp=%d component=%s level=%s source_ref=%s message=%s\n",
			i,
			record.Timestamp,
			record.Component,
			record.Level,
			record.SourceRef,
			toString(record.Message),
		)
	}
	return nil
}
