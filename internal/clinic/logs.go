package clinic

import (
	"context"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func (c *LogClient) QueryRange(ctx context.Context, query LogRangeQuery) (LogQueryResult, error) {
	target, err := c.resolveCloudLogsTarget(ctx)
	if err != nil {
		return LogQueryResult{}, err
	}
	result, err := c.handle.client.clinic.QueryLogsRange(ctx, target, query)
	if err != nil {
		return LogQueryResult{}, err
	}
	return streamResultFromLoki(result), nil
}
func (c *LogClient) Labels(ctx context.Context, query LogLabelsQuery) (LogLabelsResult, error) {
	target, err := c.resolveCloudLogsTarget(ctx)
	if err != nil {
		return LogLabelsResult{}, err
	}
	result, err := c.handle.client.clinic.LogLabels(ctx, target, query)
	if err != nil {
		return LogLabelsResult{}, err
	}
	return listResultFromValues(result.Values, result.Status), nil
}
func (c *LogClient) LabelValues(ctx context.Context, query LogLabelValuesQuery) (LogLabelsResult, error) {
	target, err := c.resolveCloudLogsTarget(ctx)
	if err != nil {
		return LogLabelsResult{}, err
	}
	result, err := c.handle.client.clinic.LogLabelValues(ctx, target, query)
	if err != nil {
		return LogLabelsResult{}, err
	}
	return listResultFromValues(result.Values, result.Status), nil
}
func (c *LogClient) resolveCloudLogsTarget(ctx context.Context) (logsTarget, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return logsTarget{}, &Error{Class: ErrBackend, Message: "log client is nil"}
	}
	target, err := c.handle.requireTarget("log client")
	if err != nil {
		return logsTarget{}, err
	}
	if target.Deleted {
		return logsTarget{}, unsupportedOperationError("logs", "data-plane capability is unavailable for deleted clusters")
	}
	if target.Platform != TargetPlatformCloud || target.Cloud == nil {
		return logsTarget{}, unsupportedOperationError("logs", "live log query operations are only available for cloud targets")
	}
	return target.Cloud.Logs, nil
}
func (c *clinicServiceClient) QueryLogsRange(ctx context.Context, target logsTarget, query LogRangeQuery) (apitypes.LokiQueryResult, error) {
	result, err := c.api.QueryLogsRange(ctx, apitypes.LokiQueryRangeRequest{
		ClusterID: target.ClusterID,
		Query:     query.Query,
		Start:     query.Start,
		End:       query.End,
		Limit:     query.Limit,
		Direction: query.Direction,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) LogLabels(ctx context.Context, target logsTarget, query LogLabelsQuery) (apitypes.LokiLabelsResult, error) {
	result, err := c.api.LogLabels(ctx, apitypes.LokiLabelsRequest{
		ClusterID: target.ClusterID,
		Start:     query.Start,
		End:       query.End,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) LogLabelValues(ctx context.Context, target logsTarget, query LogLabelValuesQuery) (apitypes.LokiLabelsResult, error) {
	result, err := c.api.LogLabelValues(ctx, apitypes.LokiLabelValuesRequest{
		ClusterID: target.ClusterID,
		LabelName: query.LabelName,
		Start:     query.Start,
		End:       query.End,
	})
	return result, mapAPIError(err)
}
