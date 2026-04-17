package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"strings"
)

func (c *MetricsClient) QueryRange(ctx context.Context, query TimeSeriesQuery) (MetricQueryRangeResult, error) {
	target, err := c.resolveMetricsTarget(ctx)
	if err != nil {
		return MetricQueryRangeResult{}, err
	}
	return c.handle.client.clinic.QueryRange(ctx, target, query)
}
func (c *MetricsClient) QueryInstant(ctx context.Context, query TimeSeriesQuery) (MetricQueryInstantResult, error) {
	target, err := c.resolveMetricsTarget(ctx)
	if err != nil {
		return MetricQueryInstantResult{}, err
	}
	return c.handle.client.clinic.QueryInstant(ctx, target, query)
}
func (c *MetricsClient) QuerySeries(ctx context.Context, query TimeSeriesQuery) (MetricQuerySeriesResult, error) {
	target, err := c.resolveMetricsTarget(ctx)
	if err != nil {
		return MetricQuerySeriesResult{}, err
	}
	return c.handle.client.clinic.QuerySeries(ctx, target, query)
}
func (c *MetricsClient) SeriesExists(ctx context.Context, query TimeSeriesQuery) (bool, MetricQuerySeriesResult, error) {
	result, err := c.QuerySeries(ctx, query)
	if err != nil {
		return false, MetricQuerySeriesResult{}, err
	}
	return len(result.Series) > 0, result, nil
}
func (c *MetricsClient) resolveMetricsTarget(ctx context.Context) (metricsTarget, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return metricsTarget{}, &Error{Class: ErrBackend, Message: "metrics client is nil"}
	}
	target, err := c.handle.requireCapability(ctx, CapabilityMetrics)
	if err != nil {
		return metricsTarget{}, err
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
func (c *clinicServiceClient) QueryInstant(ctx context.Context, target metricsTarget, query TimeSeriesQuery) (MetricQueryInstantResult, error) {
	result, err := c.api.QueryInstant(ctx, apitypes.MetricsQueryInstantRequest{
		Context: target.Context,
		Query:   strings.TrimSpace(query.Query),
		Time:    query.Time,
		Timeout: query.Timeout,
	})
	if err != nil || len(result.Series) > 0 || target.ParentFallbackLabel == "" {
		return result, mapAPIError(err)
	}
	queryString, ok := rewriteClusterIDInPromQL(query.Query, target.Context.ClusterID, target.ParentFallbackLabel)
	if !ok {
		return result, nil
	}
	retry, err := c.api.QueryInstant(ctx, apitypes.MetricsQueryInstantRequest{
		Context: target.Context,
		Query:   queryString,
		Time:    query.Time,
		Timeout: query.Timeout,
	})
	return retry, mapAPIError(err)
}
func (c *clinicServiceClient) QuerySeries(ctx context.Context, target metricsTarget, query TimeSeriesQuery) (MetricQuerySeriesResult, error) {
	result, err := c.api.QuerySeries(ctx, apitypes.MetricsQuerySeriesRequest{
		Context: target.Context,
		Match:   append([]string(nil), query.Match...),
		Start:   query.Start,
		End:     query.End,
		Timeout: query.Timeout,
	})
	if err != nil || len(result.Series) > 0 || target.ParentFallbackLabel == "" {
		return result, mapAPIError(err)
	}
	matches, ok := rewriteClusterIDMatchers(query.Match, target.Context.ClusterID, target.ParentFallbackLabel)
	if !ok {
		return result, nil
	}
	retry, err := c.api.QuerySeries(ctx, apitypes.MetricsQuerySeriesRequest{
		Context: target.Context,
		Match:   matches,
		Start:   query.Start,
		End:     query.End,
		Timeout: query.Timeout,
	})
	return retry, mapAPIError(err)
}
