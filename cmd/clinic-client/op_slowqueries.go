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

func newOPSlowQueriesCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "slowqueries",
		Short: "Query On-Premise (OP) slow queries with automatic item selection",
		Long:  opWorkflowHelp("Query On-Premise (OP) slow queries with automatic item selection."),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deps.runOPSlowQueriesQuery()
		},
	}
}

func runOPSlowQueriesQuery(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	cfg, err := loadOPItemRequestConfig(lookup, now)
	if err != nil {
		return err
	}
	client, err := newSDKClient(cfg.Base, logger)
	if err != nil {
		return err
	}
	item, err := resolveOPCatalogItem(context.Background(), client, cfg.Base, opCatalogIntentSlowQueries)
	if err != nil {
		return err
	}
	result, err := client.SlowQueries.Query(context.Background(), clinicapi.SlowQueryRequest{
		Context:   cfg.Base.Context,
		ItemID:    item.ItemID,
		StartTime: cfg.Base.Start,
		EndTime:   cfg.Base.End,
		OrderBy:   cfg.OrderBy,
		Desc:      cfg.Desc,
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
			"slowquery[%d] digest=%s query_time=%f exec_count=%d db=%s user=%s source_ref=%s query=%s\n",
			i,
			record.Digest,
			record.QueryTime,
			record.ExecCount,
			record.DB,
			record.User,
			record.SourceRef,
			toString(record.SQLText),
		)
	}
	return nil
}
