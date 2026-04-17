package cli

import (
	"context"
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func newSDKClient(cfg cliConfig, logger *log.Logger) (*clinicapi.Client, error) {
	clientCfg := clinicapi.Config{
		BaseURL:            cfg.BaseURL,
		BearerToken:        cfg.APIKey,
		Timeout:            cfg.Timeout,
		Logger:             logger,
		VerboseRequestLogs: cfg.VerboseLogs,
	}
	if cfg.RebuildProbeInterval > 0 {
		clientCfg.RebuildProbeInterval = cfg.RebuildProbeInterval
	}
	return clinicapi.NewClientWithConfig(clientCfg)
}
func withSDKClient(ctx context.Context, cfg cliConfig, logger *log.Logger, run func(context.Context, *clinicapi.Client) error) error {
	client, err := newSDKClient(cfg, logger)
	if err != nil {
		return err
	}
	return run(ctx, client)
}
func withResolvedCluster(ctx context.Context, cfg cliConfig, logger *log.Logger, run func(context.Context, *clinicapi.ClusterHandle) error) error {
	return withSDKClient(ctx, cfg, logger, func(ctx context.Context, client *clinicapi.Client) error {
		cluster, err := cfg.resolveHandle(ctx, client)
		if err != nil {
			return err
		}
		return run(ctx, cluster)
	})
}
func identityConfig(cfg cliConfig) cliConfig { return cfg }
func runLoadedClient[C any](
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
	load func(func(string) (string, bool), func() time.Time) (C, error),
	base func(C) cliConfig,
	run func(context.Context, *clinicapi.Client, C, io.Writer) error,
) error {
	cfg, err := load(lookup, now)
	if err != nil {
		return err
	}
	return withSDKClient(context.Background(), base(cfg), logger, func(ctx context.Context, client *clinicapi.Client) error {
		return run(ctx, client, cfg, out)
	})
}
func runLoadedCluster[C any](
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
	load func(func(string) (string, bool), func() time.Time) (C, error),
	base func(C) cliConfig,
	run func(context.Context, *clinicapi.ClusterHandle, C, io.Writer) error,
) error {
	cfg, err := load(lookup, now)
	if err != nil {
		return err
	}
	return withResolvedCluster(context.Background(), base(cfg), logger, func(ctx context.Context, cluster *clinicapi.ClusterHandle) error {
		return run(ctx, cluster, cfg, out)
	})
}
func runMetricsQueryRange(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadMetricsQueryConfig, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Metrics.QueryRange(ctx, clinicapi.TimeSeriesQuery{
			Query: cfg.Query,
			Start: cfg.Start,
			End:   cfg.End,
			Step:  cfg.Step,
		})
		if err != nil {
			return err
		}
		writeMetricQueryRangeSummary(out, cfg.Query, cfg.Start, cfg.End, cfg.Step, result)
		return nil
	})
}
func runMetricsQueryInstant(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadMetricsQueryConfig, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Metrics.QueryInstant(ctx, clinicapi.TimeSeriesQuery{
			Query: cfg.Query,
			Time:  cfg.Time,
		})
		if err != nil {
			return err
		}
		writeMetricQueryInstantSummary(out, cfg.Query, cfg.Time, result)
		return nil
	})
}
func runMetricsQuerySeries(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadMetricsSeriesConfig, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Metrics.QuerySeries(ctx, clinicapi.TimeSeriesQuery{
			Match: cfg.Match,
			Start: cfg.Start,
			End:   cfg.End,
		})
		if err != nil {
			return err
		}
		writeMetricQuerySeriesSummary(out, cfg.Match, cfg.Start, cfg.End, result)
		return nil
	})
}
func runClusterDetail(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		detail, err := cluster.Detail(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "cluster_id=%s\n", detail.Cluster.ClusterID)
		fmt.Fprintf(out, "detail_id=%s\n", detail.ID)
		fmt.Fprintf(out, "name=%s\n", detail.Name)
		fmt.Fprintf(out, "topology=%s\n", detail.Topology())
		return nil
	})
}
func runClusterSearch(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedClient(lookup, now, logger, out, loadClusterSearchConfig, func(cfg cloudClusterSearchConfig) cliConfig { return cfg.Base }, func(ctx context.Context, client *clinicapi.Client, cfg cloudClusterSearchConfig, out io.Writer) error {
		items, err := client.Clusters.Search(ctx, clinicapi.ClusterSearchQuery{
			Query:       cfg.Query,
			ClusterID:   cfg.Base.ClusterID,
			ShowDeleted: cfg.ShowDeleted,
			Limit:       cfg.Limit,
			Page:        cfg.Page,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "clusters=%d\n", len(items))
		for i, item := range items {
			fmt.Fprintf(out, "cluster[%d] id=%s name=%s org_id=%s tenant_id=%s cluster_type=%s provider=%s region=%s deploy_type=%s deploy_type_v2=%s parent_id=%s status=%s\n",
				i, item.ClusterID, item.Name, item.OrgID, item.TenantID, item.ClusterType, item.Provider, item.Region, item.DeployType, item.DeployTypeV2, item.ParentID, item.Status)
		}
		return nil
	})
}
func runClusterTopology(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		topology, err := cluster.Topology(ctx)
		if err != nil {
			return err
		}
		deployType := topology.Cluster.DeployTypeV2
		if deployType == "" {
			deployType = topology.Cluster.DeployType
		}
		fmt.Fprintf(out, "cluster_id=%s\n", topology.Cluster.ClusterID)
		fmt.Fprintf(out, "deploy_type=%s\n", deployType)
		fmt.Fprintf(out, "topology_id=%s\n", topology.ID)
		fmt.Fprintf(out, "topology_name=%s\n", topology.Name)
		fmt.Fprintf(out, "topology=%s\n", topology.Topology())
		return nil
	})
}
func runCloudEventsQuery(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Events(ctx, cfg.Start, cfg.End)
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
	})
}
func runCloudEventsDetail(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudEventDetailConfig, func(cfg cloudEventDetailConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudEventDetailConfig, out io.Writer) error {
		result, err := cluster.EventDetail(ctx, cfg.EventID)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "event_id=%s\n", cfg.EventID)
		fmt.Fprintf(out, "detail=%v\n", result.Detail)
		return nil
	})
}
func runCloudLokiQuery(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudLokiQueryConfig, func(cfg cloudLokiQueryConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudLokiQueryConfig, out io.Writer) error {
		result, err := cluster.Logs.Query(ctx, clinicapi.LogQuery{
			Query:     cfg.Query,
			Time:      cfg.Base.Time,
			Limit:     cfg.Limit,
			Direction: cfg.Direction,
		})
		if err != nil {
			return err
		}
		writeLokiSummary(out, result)
		return nil
	})
}
func runCloudLokiQueryRange(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudLokiQueryConfig, func(cfg cloudLokiQueryConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudLokiQueryConfig, out io.Writer) error {
		result, err := cluster.Logs.QueryRange(ctx, clinicapi.LogRangeQuery{
			Query:     cfg.Query,
			Start:     cfg.Base.Start,
			End:       cfg.Base.End,
			Limit:     cfg.Limit,
			Direction: cfg.Direction,
		})
		if err != nil {
			return err
		}
		writeLokiSummary(out, result)
		return nil
	})
}
func runCloudLokiLabels(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Logs.Labels(ctx, clinicapi.LogLabelsQuery{
			Start: cfg.Start,
			End:   cfg.End,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "labels=%v\n", listValues(result))
		return nil
	})
}
func runCloudLokiLabelValues(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudLokiLabelValuesConfig, func(cfg cloudLokiLabelValuesConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudLokiLabelValuesConfig, out io.Writer) error {
		result, err := cluster.Logs.LabelValues(ctx, clinicapi.LogLabelValuesQuery{
			LabelName: cfg.LabelName,
			Start:     cfg.Base.Start,
			End:       cfg.Base.End,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "label=%s values=%v\n", cfg.LabelName, listValues(result))
		return nil
	})
}
func runRetainedLogsSearch(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	return runLoadedCluster(lookup, now, logger, out, loadRetainedItemRequestConfig, func(cfg retainedItemRequestConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg retainedItemRequestConfig, out io.Writer) error {
		result, err := cluster.Logs.Search(ctx, clinicapi.LogSearchQuery{
			StartTime: cfg.Base.Start,
			EndTime:   cfg.Base.End,
			Pattern:   cfg.Pattern,
			Limit:     cfg.Limit,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "total=%d\n", result.Total)
		for i, record := range result.Items {
			fmt.Fprintf(
				out,
				"log[%d] timestamp=%s component=%s level=%s source_ref=%s message=%s\n",
				i,
				toString(record["timestamp"]),
				toString(record["component"]),
				toString(record["level"]),
				firstNonEmptyString(record["source_ref"], record["item_id"]),
				toString(record["message"]),
			)
		}
		return nil
	})
}
func runCloudDataProxyQuery(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudDataProxyQueryConfig, func(cfg cloudDataProxyQueryConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudDataProxyQueryConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.Query(ctx, clinicapi.SQLQuery{
			SQL:     cfg.SQL,
			Timeout: cfg.Timeout,
		})
		if err != nil {
			return err
		}
		writeTableResult(out, result)
		return nil
	})
}
func runCloudDataProxySchema(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudDataProxySchemaConfig, func(cfg cloudDataProxySchemaConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudDataProxySchemaConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.Schema(ctx, clinicapi.SchemaQuery{
			Tables: cfg.Tables,
		})
		if err != nil {
			return err
		}
		writeTableResult(out, result)
		return nil
	})
}
func runCapabilitySQLStatements(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudDataProxyQueryConfig, func(cfg cloudDataProxyQueryConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudDataProxyQueryConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.SQLStatements(ctx, clinicapi.SQLStatementsQuery{
			SQL:     cfg.SQL,
			Timeout: cfg.Timeout,
			Start:   strconv.FormatInt(cfg.Base.Start, 10),
			End:     strconv.FormatInt(cfg.Base.End, 10),
		})
		if err != nil {
			return err
		}
		writeTableResult(out, result)
		return nil
	})
}
func runCloudTopSQLSummary(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudTopSQLConfig, func(cfg cloudTopSQLConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudTopSQLConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.TopSQLSummary(ctx, clinicapi.TopSQLSummaryQuery{
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
		fmt.Fprintf(out, "total=%d\n", len(result.Rows))
		for i, row := range result.Rows {
			fmt.Fprintf(
				out,
				"topsql[%d] digest=%s cpu_time_ms=%f exec_count_per_sec=%f duration_per_exec_ms=%f query=%s\n",
				i,
				toString(row["sql_digest"]),
				toFloat64(row["cpu_time_ms"]),
				toFloat64(row["exec_count_per_sec"]),
				toFloat64(row["duration_per_exec_ms"]),
				toString(row["sql_text"]),
			)
		}
		return nil
	})
}
func runCloudTopSlowQueries(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudTopSlowQueriesConfig, func(cfg cloudTopSlowQueriesConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudTopSlowQueriesConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.TopSlowQueries(ctx, clinicapi.TopSlowQueriesQuery{
			Start:   cfg.Start,
			Hours:   cfg.Hours,
			OrderBy: cfg.OrderBy,
			Limit:   cfg.Limit,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "total=%d\n", len(result.Rows))
		for i, row := range result.Rows {
			fmt.Fprintf(
				out,
				"slowquery_top[%d] digest=%s db=%s count=%d sum_latency=%f max_latency=%f avg_latency=%f query=%s\n",
				i,
				toString(row["sql_digest"]),
				toString(row["db"]),
				toInt64(row["count"]),
				toFloat64(row["sum_latency"]),
				toFloat64(row["max_latency"]),
				toFloat64(row["avg_latency"]),
				toString(row["sql_text"]),
			)
		}
		return nil
	})
}
func runCloudSlowQueriesList(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudSlowQueryListConfig, func(cfg cloudSlowQueryListConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudSlowQueryListConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.SlowQuerySamples(ctx, clinicapi.SlowQuerySamplesQuery{
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
		fmt.Fprintf(out, "total=%d\n", len(result.Items))
		for i, row := range result.Items {
			itemID := firstNonEmptyString(row["item_id"], row["source_ref"])
			fmt.Fprintf(
				out,
				"slowquery_list[%d] id=%s item_id=%s digest=%s timestamp=%s query_time=%f connection_id=%s query=%s\n",
				i,
				toString(row["id"]),
				itemID,
				toString(row["digest"]),
				toString(row["timestamp"]),
				toFloat64(row["query_time"]),
				toString(row["connection_id"]),
				toString(row["query"]),
			)
		}
		return nil
	})
}
func runCloudSlowQueriesDetail(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudSlowQueryDetailConfig, func(cfg cloudSlowQueryDetailConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudSlowQueryDetailConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.SlowQueryDetail(ctx, clinicapi.SlowQueryDetailQuery{
			ID:           cfg.ID,
			Start:        strconv.FormatInt(cfg.Base.Start, 10),
			End:          strconv.FormatInt(cfg.Base.End, 10),
			Digest:       cfg.Digest,
			ConnectionID: cfg.ConnectionID,
			Timestamp:    cfg.Timestamp,
		})
		if err != nil {
			return err
		}
		itemID := firstNonEmptyString(result.Fields["item_id"], result.Fields["source_ref"])
		writeOptionalKeyValue(out, "id", toString(result.Fields["id"]))
		writeOptionalKeyValue(out, "item_id", itemID)
		writeOptionalKeyValue(out, "digest", toString(result.Fields["digest"]))
		writeOptionalKeyValue(out, "connection_id", toString(result.Fields["connection_id"]))
		writeOptionalKeyValue(out, "timestamp", toString(result.Fields["timestamp"]))
		fmt.Fprintf(out, "detail=%v\n", result.Fields)
		return nil
	})
}
func runRetainedSlowQueriesQuery(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	return runLoadedCluster(lookup, now, logger, out, loadRetainedItemRequestConfig, func(cfg retainedItemRequestConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg retainedItemRequestConfig, out io.Writer) error {
		result, err := cluster.SQLAnalytics.SlowQueryRecords(ctx, clinicapi.SlowQueryRecordsQuery{
			StartTime: cfg.Base.Start,
			EndTime:   cfg.Base.End,
			OrderBy:   cfg.OrderBy,
			Desc:      cfg.Desc,
			Limit:     cfg.Limit,
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "request_window=%d..%d order_by=%s desc=%t limit=%d\n", cfg.Base.Start, cfg.Base.End, cfg.OrderBy, cfg.Desc, cfg.Limit)
		fmt.Fprintf(out, "total=%d\n", len(result.Rows))
		for i, record := range result.Rows {
			fmt.Fprintf(
				out,
				"slowquery[%d] digest=%s query_time=%f exec_count=%d db=%s user=%s source_ref=%s query=%s\n",
				i,
				toString(record["digest"]),
				toFloat64(record["query_time"]),
				toInt64(record["exec_count"]),
				toString(record["db"]),
				toString(record["user"]),
				firstNonEmptyString(record["source_ref"], record["item_id"]),
				toString(record["sql_text"]),
			)
		}
		return nil
	})
}
func runRetainedConfigsGet(lookup func(string) (string, bool), now func() time.Time, logger *log.Logger, out io.Writer) error {
	return runLoadedCluster(lookup, now, logger, out, loadRetainedItemRequestConfig, func(cfg retainedItemRequestConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg retainedItemRequestConfig, out io.Writer) error {
		result, err := cluster.Configs.Get(ctx, clinicapi.ConfigQuery{})
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "total=%d\n", len(result.Rows))
		for i, entry := range result.Rows {
			fmt.Fprintf(
				out,
				"config[%d] component=%s key=%s value=%s source_ref=%s\n",
				i,
				toString(entry["component"]),
				toString(entry["key"]),
				toString(entry["value"]),
				firstNonEmptyString(entry["source_ref"], entry["item_id"]),
			)
		}
		return nil
	})
}
func runCloudProfilingGroups(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Profiling.ListGroups(ctx, cfg.Start, cfg.End)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "groups=%d\n", len(result.Items))
		for i, group := range result.Items {
			fmt.Fprintf(out, "group[%d] ts=%d state=%s duration=%d component_num=%v\n", i, toInt64(group["timestamp"]), toString(group["state"]), toInt64(group["profile_duration_secs"]), group["component_num"])
		}
		return nil
	})
}
func runCloudProfilingDetail(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudProfilingDetailConfig, func(cfg cloudProfilingDetailConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudProfilingDetailConfig, out io.Writer) error {
		result, err := cluster.Profiling.Detail(ctx, cfg.Timestamp)
		if err != nil {
			return err
		}
		items := objectItems(result.Fields, "target_profiles")
		fmt.Fprintf(out, "target_profiles=%d\n", len(items))
		for i, item := range items {
			fmt.Fprintf(out, "target_profile[%d] component=%s address=%s profile_type=%s state=%s error=%s\n", i, toString(item["component"]), toString(item["address"]), toString(item["profile_type"]), toString(item["state"]), toString(item["error"]))
		}
		return nil
	})
}
func runCloudProfilingDownload(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudProfilingDownloadConfig, func(cfg cloudProfilingDownloadConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudProfilingDownloadConfig, out io.Writer) error {
		request := clinicapi.ProfileFetchRequest{
			Timestamp:   cfg.Timestamp,
			ProfileType: cfg.ProfileType,
			Component:   cfg.Component,
			Address:     cfg.Address,
			DataFormat:  cfg.DataFormat,
		}
		artifact, err := cluster.Profiling.Fetch(ctx, request)
		if err != nil {
			return err
		}
		outputPath := outputPathOrDefault(cfg.OutputPath, filepath.Join(".", defaultProfileFilename(request)))
		return writeArtifact(out, outputPath, artifact)
	})
}
func runCloudDiagnosticPlan(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runCloudDiagnosticList(lookup, now, logger, out, "plan")
}
func runCloudDiagnosticOOM(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runCloudDiagnosticList(lookup, now, logger, out, "oom")
}
func runCloudDiagnosticList(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
	kind string,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		var (
			result clinicapi.DiagnosticListResult
			err    error
		)
		switch kind {
		case "plan":
			result, err = cluster.Diagnostics.ListPlanReplayer(ctx, cfg.Start, cfg.End)
		default:
			result, err = cluster.Diagnostics.ListOOMRecord(ctx, cfg.Start, cfg.End)
		}
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "records=%d\n", len(result.Items))
		for i, record := range result.Items {
			files := anySliceAsMaps(record["files"])
			fmt.Fprintf(out, "record[%d] files=%d\n", i, len(files))
			for j, file := range files {
				fmt.Fprintf(out, "record[%d].file[%d] name=%s key=%s size=%d download_url=%s\n", i, j, toString(file["name"]), toString(file["key"]), toInt64(file["size"]), toString(file["download_url"]))
			}
		}
		return nil
	})
}
func runCloudDiagnosticDownload(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudDiagnosticDownloadConfig, func(cfg cloudDiagnosticDownloadConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudDiagnosticDownloadConfig, out io.Writer) error {
		artifact, err := cluster.Diagnostics.Download(ctx, clinicapi.DiagnosticDownloadRequest{
			Key: cfg.Key,
		})
		if err != nil {
			return err
		}
		outputPath := outputPathOrDefault(cfg.OutputPath, filepath.Join(".", defaultDiagnosticFilename(cfg.Key)))
		return writeArtifact(out, outputPath, artifact)
	})
}
func runCapabilityDiscover(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Capabilities.Discover(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "cluster_id=%s\n", result.Cluster.ClusterID)
		deployType := result.Cluster.DeployTypeV2
		if strings.TrimSpace(deployType) == "" {
			deployType = result.Cluster.DeployType
		}
		fmt.Fprintf(out, "deploy_type=%s\n", deployType)
		fmt.Fprintf(out, "deleted=%t\n", result.Cluster.Deleted)
		fmt.Fprintf(out, "capabilities=%d\n", len(result.Capabilities))
		for i, item := range result.Capabilities {
			fmt.Fprintf(
				out,
				"capability[%d] name=%s available=%t scope=%s stability=%s requires_parent_target=%t requires_live_cluster=%t tier_constraints=%s reason=%s\n",
				i,
				item.Name,
				item.Available,
				item.Scope,
				item.Stability,
				item.RequiresParentTarget,
				item.RequiresLiveCluster,
				strings.Join(item.TierConstraints, ","),
				item.Reason,
			)
		}
		return nil
	})
}
func writeLokiSummary(out io.Writer, result clinicapi.LogQueryResult) {
	fmt.Fprintf(out, "status=%s result_type=streams streams=%d\n", result.Status, len(result.Streams))
	for i, stream := range result.Streams {
		fmt.Fprintf(out, "stream[%d] labels=%v values=%d\n", i, stream.Labels, len(stream.Values))
	}
}
func writeTableResult(out io.Writer, result clinicapi.TableResult) {
	fmt.Fprintf(out, "columns=%v\n", result.Columns)
	fmt.Fprintf(out, "rows=%d\n", len(result.Rows))
	fmt.Fprintf(
		out,
		"metadata row_count=%d bytes_scanned=%d execution_time=%s query_id=%s engine=%s vendor=%s region=%s\n",
		result.Metadata.RowCount,
		result.Metadata.BytesScanned,
		result.Metadata.ExecutionTime,
		result.Metadata.QueryID,
		result.Metadata.Engine,
		result.Metadata.Vendor,
		result.Metadata.Region,
	)
	for i, row := range result.Rows {
		fmt.Fprintf(out, "row[%d]=%v\n", i, row)
	}
}
func listValues(result clinicapi.LogLabelsResult) []string {
	values := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		if value := strings.TrimSpace(toString(item["value"])); value != "" {
			values = append(values, value)
		}
	}
	return values
}
func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(toString(value)); text != "" {
			return text
		}
	}
	return ""
}
func toInt64(v any) int64 {
	switch value := v.(type) {
	case int:
		return int64(value)
	case int8:
		return int64(value)
	case int16:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case uint:
		return int64(value)
	case uint8:
		return int64(value)
	case uint16:
		return int64(value)
	case uint32:
		return int64(value)
	case uint64:
		return int64(value)
	case float32:
		return int64(value)
	case float64:
		return int64(value)
	case string:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		return parsed
	default:
		parsed, _ := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(v)), 10, 64)
		return parsed
	}
}
func toFloat64(v any) float64 {
	switch value := v.(type) {
	case float32:
		return float64(value)
	case float64:
		return value
	case int:
		return float64(value)
	case int8:
		return float64(value)
	case int16:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	case uint:
		return float64(value)
	case uint8:
		return float64(value)
	case uint16:
		return float64(value)
	case uint32:
		return float64(value)
	case uint64:
		return float64(value)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return parsed
	default:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(v)), 64)
		return parsed
	}
}
func objectItems(fields map[string]any, key string) []map[string]any {
	if fields == nil {
		return nil
	}
	return anySliceAsMaps(fields[key])
}
func anySliceAsMaps(v any) []map[string]any {
	switch value := v.(type) {
	case []map[string]any:
		return value
	case []any:
		out := make([]map[string]any, 0, len(value))
		for _, item := range value {
			if mapped, ok := item.(map[string]any); ok {
				out = append(out, mapped)
			}
		}
		return out
	default:
		return nil
	}
}
func writeOptionalKeyValue(out io.Writer, key, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	fmt.Fprintf(out, "%s=%s\n", key, trimmed)
}
func defaultDiagnosticFilename(key string) string {
	base := filepath.Base(strings.TrimSpace(key))
	if base == "" || base == "." || base == ".." {
		return "diagnostic.dat"
	}
	return base
}
func defaultProfileFilename(req clinicapi.ProfileFetchRequest) string {
	component := strings.NewReplacer("/", "_", ":", "_").Replace(strings.TrimSpace(req.Component) + "-" + strings.TrimSpace(req.Address))
	component = strings.Trim(component, "-_")
	if component == "" {
		component = "unknown"
	}
	profileType := strings.TrimSpace(req.ProfileType)
	if profileType == "" {
		profileType = "profile"
	}
	return fmt.Sprintf("pprof-%d-%s-%s.pb", req.Timestamp, component, profileType)
}
