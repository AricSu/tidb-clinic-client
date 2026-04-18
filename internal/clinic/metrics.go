package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"github.com/AricSu/tidb-clinic-client/internal/compiler"
	"strings"
)

func (c *MetricsClient) QueryRange(ctx context.Context, query TimeSeriesQuery) (MetricQueryRangeResult, error) {
	target, err := c.resolveMetricsTarget(ctx)
	if err != nil {
		return MetricQueryRangeResult{}, err
	}
	return c.handle.client.clinic.QueryRange(ctx, target, query)
}
func (c *MetricsClient) CompileRange(ctx context.Context, query MetricsCompileQuery) ([]CompiledTimeseriesDigest, error) {
	target, err := c.resolveMetricsTarget(ctx)
	if err != nil {
		return nil, err
	}
	return c.handle.client.clinic.CompileRange(ctx, target, query)
}
func (c *MetricsClient) resolveMetricsTarget(ctx context.Context) (metricsTarget, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return metricsTarget{}, &Error{Class: ErrBackend, Message: "metrics client is nil"}
	}
	target, err := c.handle.requireTarget("metrics client")
	if err != nil {
		return metricsTarget{}, err
	}
	if target.Deleted {
		return metricsTarget{}, unsupportedOperationError("metrics", "data-plane capability is unavailable for deleted clusters")
	}
	switch target.Platform {
	case TargetPlatformCloud:
		if target.Cloud == nil {
			return metricsTarget{}, &Error{Class: ErrBackend, Message: "cloud target is missing"}
		}
		return target.Cloud.Metrics, nil
	case TargetPlatformTiUPCluster:
		requestContext, ok := target.requestContext()
		if !ok {
			return metricsTarget{}, &Error{Class: ErrBackend, Message: "tiup-cluster request context is missing"}
		}
		return metricsTarget{Context: requestContext}, nil
	default:
		return metricsTarget{}, &Error{Class: ErrInvalidRequest, Message: "unsupported target platform"}
	}
}
func (c *clinicServiceClient) QueryRange(ctx context.Context, target metricsTarget, query TimeSeriesQuery) (MetricQueryRangeResult, error) {
	result, err := c.api.QueryRangeWithAutoSplit(ctx, apitypes.MetricsQueryRangeRequest{
		Context: target.Context,
		Query:   strings.TrimSpace(query.Query),
		Start:   query.Start,
		End:     query.End,
		Step:    query.Step,
		Timeout: query.Timeout,
	})
	if err != nil || len(result.Series) > 0 || target.ParentFallbackLabel == "" {
		return result, mapAPIError(err)
	}
	queryString, ok := rewriteClusterIDInPromQL(query.Query, target.Context.ClusterID, target.ParentFallbackLabel)
	if !ok {
		return result, nil
	}
	retry, err := c.api.QueryRangeWithAutoSplit(ctx, apitypes.MetricsQueryRangeRequest{
		Context: target.Context,
		Query:   queryString,
		Start:   query.Start,
		End:     query.End,
		Step:    query.Step,
		Timeout: query.Timeout,
	})
	return retry, mapAPIError(err)
}
func (c *clinicServiceClient) CompileRange(ctx context.Context, target metricsTarget, query MetricsCompileQuery) ([]CompiledTimeseriesDigest, error) {
	result, err := c.QueryRange(ctx, target, TimeSeriesQuery{
		Query:   strings.TrimSpace(query.Query),
		Start:   query.Start,
		End:     query.End,
		Step:    strings.TrimSpace(query.Step),
		Timeout: strings.TrimSpace(query.Timeout),
	})
	if err != nil {
		return nil, err
	}
	return compiler.CompileMetricQueryRangeDigests(ctx, query, result)
}
