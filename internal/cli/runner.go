package cli

import (
	"context"
	"encoding/json"
	"fmt"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	rawapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"github.com/AricSu/tidb-clinic-client/internal/compiler"
	"io"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func newSDKClient(cfg cliConfig, hooks clinicapi.Hooks) (*clinicapi.Client, error) {
	clientCfg := clinicapi.Config{
		BaseURL:     cfg.BaseURL,
		BearerToken: cfg.APIKey,
		Hooks:       hooks,
	}
	if cfg.RebuildProbeInterval > 0 {
		clientCfg.RebuildProbeInterval = cfg.RebuildProbeInterval
	}
	return clinicapi.NewClientWithConfig(clientCfg)
}
func newRawAPIClient(cfg cliConfig, hooks rawapi.Hooks) (*rawapi.Client, error) {
	clientCfg := rawapi.Config{
		BaseURL:     cfg.BaseURL,
		BearerToken: cfg.APIKey,
		Hooks:       hooks,
	}
	if cfg.RebuildProbeInterval > 0 {
		clientCfg.RebuildProbeInterval = cfg.RebuildProbeInterval
	}
	return rawapi.NewClientWithConfig(clientCfg)
}
func withSDKClient(ctx context.Context, cfg cliConfig, hooks clinicapi.Hooks, run func(context.Context, *clinicapi.Client) error) error {
	client, err := newSDKClient(cfg, hooks)
	if err != nil {
		return err
	}
	return run(ctx, client)
}
func withRawAPIClient(ctx context.Context, cfg cliConfig, hooks rawapi.Hooks, run func(context.Context, *rawapi.Client) error) error {
	client, err := newRawAPIClient(cfg, hooks)
	if err != nil {
		return err
	}
	return run(ctx, client)
}

func withResolvedCluster(ctx context.Context, cfg cliConfig, hooks clinicapi.Hooks, run func(context.Context, *clinicapi.ClusterHandle) error) error {
	return withSDKClient(ctx, cfg, hooks, func(ctx context.Context, client *clinicapi.Client) error {
		cluster, err := cfg.resolveHandle(ctx, client)
		if err != nil {
			return err
		}
		return run(ctx, cluster)
	})
}

func withResolvedClusterAndRawClient(
	ctx context.Context,
	cfg cliConfig,
	sdkHooks clinicapi.Hooks,
	rawHooks rawapi.Hooks,
	run func(context.Context, *clinicapi.ClusterHandle, *rawapi.Client) error,
) error {
	return withResolvedCluster(ctx, cfg, sdkHooks, func(ctx context.Context, cluster *clinicapi.ClusterHandle) error {
		return withRawAPIClient(ctx, cfg, rawHooks, func(ctx context.Context, rawClient *rawapi.Client) error {
			return run(ctx, cluster, rawClient)
		})
	})
}
func requireCloudCluster(cluster *clinicapi.ClusterHandle, endpoint, capability string) error {
	if cluster == nil {
		return &clinicapi.Error{
			Class:    clinicapi.ErrBackend,
			Endpoint: endpoint,
			Message:  "resolved cluster is nil",
		}
	}
	if cluster.Platform() == clinicapi.TargetPlatformCloud {
		return nil
	}
	return &clinicapi.Error{
		Class:    clinicapi.ErrUnsupported,
		Endpoint: endpoint,
		Message:  strings.TrimSpace(capability) + " are only available for cloud clusters; non-cloud deployments are not supported",
	}
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
	return withSDKClient(context.Background(), base(cfg), clinicapi.Hooks{}, func(ctx context.Context, client *clinicapi.Client) error {
		return run(ctx, client, cfg, out)
	})
}

func runLoadedRawClient[C any](
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
	load func(func(string) (string, bool), func() time.Time) (C, error),
	base func(C) cliConfig,
	run func(context.Context, *rawapi.Client, C, io.Writer) error,
) error {
	cfg, err := load(lookup, now)
	if err != nil {
		return err
	}
	return withRawAPIClient(context.Background(), base(cfg), rawapi.Hooks{}, func(ctx context.Context, client *rawapi.Client) error {
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
	return runLoadedClusterWithHooks(lookup, now, logger, out, clinicapi.Hooks{}, load, base, run)
}
func runLoadedClusterWithHooks[C any](
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
	hooks clinicapi.Hooks,
	load func(func(string) (string, bool), func() time.Time) (C, error),
	base func(C) cliConfig,
	run func(context.Context, *clinicapi.ClusterHandle, C, io.Writer) error,
) error {
	cfg, err := load(lookup, now)
	if err != nil {
		return err
	}
	return withResolvedCluster(context.Background(), base(cfg), hooks, func(ctx context.Context, cluster *clinicapi.ClusterHandle) error {
		return run(ctx, cluster, cfg, out)
	})
}

func runLoadedClusterAndRawClient[C any](
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
	load func(func(string) (string, bool), func() time.Time) (C, error),
	base func(C) cliConfig,
	run func(context.Context, *clinicapi.ClusterHandle, *rawapi.Client, C, io.Writer) error,
) error {
	cfg, err := load(lookup, now)
	if err != nil {
		return err
	}
	return withResolvedClusterAndRawClient(context.Background(), base(cfg), clinicapi.Hooks{}, rawapi.Hooks{}, func(ctx context.Context, cluster *clinicapi.ClusterHandle, rawClient *rawapi.Client) error {
		return run(ctx, cluster, rawClient, cfg, out)
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
		return writeJSON(out, result)
	})
}
func runMetricsCompile(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadMetricsCompileConfig, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		result, err := cluster.Metrics.QueryRange(ctx, clinicapi.TimeSeriesQuery{
			Query: cfg.Query,
			Start: cfg.Start,
			End:   cfg.End,
			Step:  cfg.Step,
		})
		if err != nil {
			return err
		}
		compiled, err := compiler.CompileMetricQueryRange(ctx, clinicapi.MetricsCompileQuery{
			Query: cfg.Query,
			Start: cfg.Start,
			End:   cfg.End,
			Step:  cfg.Step,
		}, result)
		if err != nil {
			return err
		}
		var decoded any
		if err := json.Unmarshal(compiled, &decoded); err != nil {
			_, writeErr := fmt.Fprintln(out, string(compiled))
			return writeErr
		}
		return writeJSON(out, decoded)
	})
}
func runSlowQuery(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	progress := newRetainedSlowQueriesProgress(logger)
	defer progress.Close()
	return runLoadedClusterWithHooks(lookup, now, logger, out, progress.Hooks(), loadSlowQueryConfig, func(cfg slowQueryConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg slowQueryConfig, out io.Writer) error {
		if strings.TrimSpace(cfg.Digest) != "" {
			result, err := cluster.SlowQueries.Samples(ctx, clinicapi.SlowQuerySamplesQuery{
				Digest:  cfg.Digest,
				Start:   strconv.FormatInt(cfg.Base.Start, 10),
				End:     strconv.FormatInt(cfg.Base.End, 10),
				OrderBy: cfg.OrderBy,
				Limit:   cfg.Limit,
				Desc:    cfg.Desc,
				Fields:  append([]string(nil), cfg.Fields...),
			})
			if err != nil {
				return err
			}
			return writeJSON(out, result)
		}
		result, err := cluster.SlowQueries.Query(ctx, clinicapi.SlowQueryQuery{
			Start:   cfg.Base.Start,
			End:     cfg.Base.End,
			OrderBy: cfg.OrderBy,
			Desc:    cfg.Desc,
			Limit:   cfg.Limit,
		})
		if err != nil {
			return err
		}
		return writeJSON(out, result)
	})
}
func runCollectedDataDownload(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	cfg, err := loadCollectedDataDownloadConfig(lookup, now)
	if err != nil {
		return err
	}
	progress := newDataDownloadProgress(logger)
	defer progress.Close()
	return withResolvedCluster(context.Background(), cfg.Base, progress.Hooks(), func(ctx context.Context, cluster *clinicapi.ClusterHandle) error {
		artifact, err := cluster.CollectedData.Download(ctx, clinicapi.CollectedDataDownloadRequest{
			StartTime: cfg.Base.Start,
			EndTime:   cfg.Base.End,
		})
		if err != nil {
			return err
		}
		outputPath := outputPathOrDefault(cfg.OutputPath, filepath.Join(".", firstNonEmptyString(artifact.Filename, "collected-data")))
		return writeArtifact(out, outputPath, artifact)
	})
}
func runCollectedDataList(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cliConfig, out io.Writer) error {
		items, err := cluster.CollectedData.List(ctx)
		if err != nil {
			return err
		}
		return writeJSON(out, struct {
			Total int                           `json:"total"`
			Items []clinicapi.CollectedDataItem `json:"items,omitempty"`
		}{
			Total: len(items),
			Items: items,
		})
	})
}
func runClusterInfo(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedRawClient(lookup, now, logger, out, loadConfigFromEnv, identityConfig, func(ctx context.Context, rawClient *rawapi.Client, cfg cliConfig, out io.Writer) error {
		raw, err := rawClient.GetClusterDetailRaw(ctx, rawapi.CloudClusterDetailRequest{
			Target: rawapi.CloudTarget{
				OrgID:     cfg.OrgID,
				ClusterID: cfg.ClusterID,
			},
		})
		if err != nil {
			return err
		}
		return writeJSON(out, raw)
	})
}
func runCloudEventsQuery(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedClusterAndRawClient(lookup, now, logger, out, loadCloudEventListConfig, func(cfg cloudEventListConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, rawClient *rawapi.Client, cfg cloudEventListConfig, out io.Writer) error {
		if err := requireCloudCluster(cluster, "cloud-events.search", "events"); err != nil {
			return err
		}
		raw, err := rawClient.QueryEventsRaw(ctx, rawapi.CloudEventsRequest{
			Target: rawapi.CloudTarget{
				OrgID:     cluster.OrgID(),
				ClusterID: cluster.ClusterID(),
			},
			StartTime: cfg.Base.Start,
			EndTime:   cfg.Base.End,
			Name:      cfg.Name,
			Severity:  cfg.Severity,
		})
		if err != nil {
			return err
		}
		return writeJSON(out, raw)
	})
}

func runCloudLogs(
	lookup func(string) (string, bool),
	now func() time.Time,
	logger *log.Logger,
	out io.Writer,
) error {
	return runLoadedCluster(lookup, now, logger, out, loadCloudLogsConfig, func(cfg cloudLogsConfig) cliConfig { return cfg.Base }, func(ctx context.Context, cluster *clinicapi.ClusterHandle, cfg cloudLogsConfig, out io.Writer) error {
		switch cfg.Mode {
		case cloudLogsModeLabelValues:
			result, err := cluster.Logs.LabelValues(ctx, clinicapi.LogLabelValuesQuery{
				LabelName: cfg.LabelName,
				Start:     cfg.Base.Start,
				End:       cfg.Base.End,
			})
			if err != nil {
				return err
			}
			return writeJSON(out, result)
		case cloudLogsModeQueryRange:
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
			return writeJSON(out, result)
		default:
			result, err := cluster.Logs.Labels(ctx, clinicapi.LogLabelsQuery{
				Start: cfg.Base.Start,
				End:   cfg.Base.End,
			})
			if err != nil {
				return err
			}
			return writeJSON(out, result)
		}
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
		return writeJSON(out, result)
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
		return writeJSON(out, result)
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
func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		if text := strings.TrimSpace(toString(value)); text != "" {
			return text
		}
	}
	return ""
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
