package clinic

import (
	"context"
	"fmt"
	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"strconv"
	"strings"
	"time"
)

const dataProxyQueryEndpoint = "/data-proxy/query"

func (c *SQLAnalyticsClient) Query(ctx context.Context, query SQLQuery) (AnalyticalResult, error) {
	target, err := c.resolveSQLTarget(ctx, CapabilitySQLQuery)
	if err != nil {
		return AnalyticalResult{}, err
	}
	result, err := c.handle.client.clinic.QuerySQL(ctx, target, query)
	if err != nil {
		return AnalyticalResult{}, err
	}
	return tableResultFromDataProxy(result), nil
}
func (c *SQLAnalyticsClient) Schema(ctx context.Context, query SchemaQuery) (SchemaResult, error) {
	target, err := c.resolveSQLTarget(ctx, CapabilitySchema)
	if err != nil {
		return SchemaResult{}, err
	}
	result, err := c.handle.client.clinic.SchemaSQL(ctx, target, query)
	if err != nil {
		return SchemaResult{}, err
	}
	return tableResultFromSchema(result), nil
}
func (c *SQLAnalyticsClient) TopSQLSummary(ctx context.Context, query TopSQLSummaryQuery) (TopSQLSummaryResult, error) {
	target, err := c.resolveCloudTarget(ctx, CapabilityTopSQL)
	if err != nil {
		return TopSQLSummaryResult{}, err
	}
	result, err := c.handle.client.clinic.TopSQLSummary(ctx, target, query)
	if err != nil {
		return TopSQLSummaryResult{}, err
	}
	return tableResultFromTopSQL(result), nil
}
func (c *SQLAnalyticsClient) TopSlowQueries(ctx context.Context, query TopSlowQueriesQuery) (SlowQuerySummaryResult, error) {
	target, err := c.resolveCloudTarget(ctx, CapabilitySlowQuery)
	if err != nil {
		return SlowQuerySummaryResult{}, err
	}
	result, err := c.handle.client.clinic.TopSlowQueries(ctx, target, query)
	if err != nil {
		return SlowQuerySummaryResult{}, err
	}
	return tableResultFromTopSlow(result), nil
}
func (c *SQLAnalyticsClient) SlowQuerySamples(ctx context.Context, query SlowQuerySamplesQuery) (SlowQuerySamplesResult, error) {
	target, err := c.resolveSlowQueryTarget(ctx)
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	result, err := c.handle.client.clinic.SlowQuerySamples(ctx, target, query)
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	return listResultFromSlowQuerySamples(result), nil
}
func (c *SQLAnalyticsClient) SlowQueryDetail(ctx context.Context, query SlowQueryDetailQuery) (SlowQueryDetail, error) {
	target, err := c.resolveSlowQueryTarget(ctx)
	if err != nil {
		return SlowQueryDetail{}, err
	}
	result, err := c.handle.client.clinic.SlowQueryDetail(ctx, target, query)
	if err != nil {
		return SlowQueryDetail{}, err
	}
	detail := cloneAnyMap(result)
	if detail == nil {
		detail = map[string]any{}
	}
	id := firstNonEmpty(strings.TrimSpace(query.ID), stringifyAny(detail["id"]))
	digest := firstNonEmpty(strings.TrimSpace(query.Digest), stringifyAny(detail["digest"]))
	connectionID := firstNonEmpty(
		strings.TrimSpace(query.ConnectionID),
		stringifyAny(detail["connection_id"]),
		stringifyAny(detail["connect_id"]),
	)
	timestamp := firstNonEmpty(strings.TrimSpace(query.Timestamp), stringifyAny(detail["timestamp"]))
	itemID := firstNonEmpty(stringifyAny(detail["item_id"]), stringifyAny(detail["itemID"]), stringifyAny(detail["source_ref"]))
	if id != "" {
		detail["id"] = id
	}
	if digest != "" {
		detail["digest"] = digest
	}
	if connectionID != "" {
		detail["connection_id"] = connectionID
	}
	if timestamp != "" {
		detail["timestamp"] = timestamp
	}
	if itemID != "" {
		detail["item_id"] = itemID
		detail["source_ref"] = itemID
	}
	return SlowQueryDetail{Fields: detail}, nil
}
func (c *SQLAnalyticsClient) SQLStatements(ctx context.Context, query SQLStatementsQuery) (AnalyticalResult, error) {
	target, err := c.resolveSQLTarget(ctx, CapabilitySQLStatements)
	if err != nil {
		return AnalyticalResult{}, err
	}
	result, err := c.handle.client.clinic.SQLStatements(ctx, target, query)
	if err != nil {
		return AnalyticalResult{}, err
	}
	return tableResultFromDataProxy(result), nil
}
func (c *SQLAnalyticsClient) SlowQueryRecords(ctx context.Context, query SlowQueryRecordsQuery) (SlowQueryRecordsResult, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return SlowQueryRecordsResult{}, &Error{Class: ErrBackend, Message: "sql analytics client is nil"}
	}
	target, err := c.handle.requireCapability(ctx, CapabilitySlowQuery)
	if err != nil {
		return SlowQueryRecordsResult{}, err
	}
	requestContext, ok := target.requestContext()
	if !ok {
		return SlowQueryRecordsResult{}, &Error{Class: ErrBackend, Message: "collected-data request context is missing"}
	}
	if target.Platform != TargetPlatformTiUPCluster {
		return SlowQueryRecordsResult{}, unsupportedOperationError("capability:slow_query.records", "slow query records are only available for tiup-cluster collected data")
	}
	itemID, err := c.handle.client.resolveCatalogItemID(ctx, target, catalogIntentSlowQueries, query.StartTime, query.EndTime)
	if err != nil {
		return SlowQueryRecordsResult{}, err
	}
	result, err := c.handle.client.clinic.SlowQueryRecords(ctx, requestContext, itemID, query)
	if err != nil {
		return SlowQueryRecordsResult{}, err
	}
	return tableResultFromSlowQueryRecords(result), nil
}
func (c *SQLAnalyticsClient) resolveSQLTarget(ctx context.Context, capability CapabilityName) (sqlTarget, error) {
	target, err := c.resolveCloudTarget(ctx, capability)
	if err != nil {
		return sqlTarget{}, err
	}
	return target.SQL, nil
}
func (c *SQLAnalyticsClient) resolveCloudTarget(ctx context.Context, capability CapabilityName) (resolvedClusterTarget, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return resolvedClusterTarget{}, &Error{Class: ErrBackend, Message: "sql analytics client is nil"}
	}
	return c.handle.resolveCloudTarget(ctx, capability)
}
func (c *SQLAnalyticsClient) resolveSlowQueryTarget(ctx context.Context) (resolvedClusterTarget, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return resolvedClusterTarget{}, &Error{Class: ErrBackend, Message: "sql analytics client is nil"}
	}
	target, err := c.handle.requireCapability(ctx, CapabilitySlowQuery)
	if err != nil {
		return resolvedClusterTarget{}, err
	}
	if target.Cloud == nil {
		return resolvedClusterTarget{}, unsupportedOperationError("capability:slow_query", "slow query capability is unavailable for the resolved cluster")
	}
	return *target.Cloud, nil
}
func (c *clinicServiceClient) QuerySQL(ctx context.Context, target sqlTarget, query SQLQuery) (apitypes.DataProxyQueryResult, error) {
	sql := strings.TrimSpace(query.SQL)
	if sql == "" {
		return apitypes.DataProxyQueryResult{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: dataProxyQueryEndpoint,
			Message:  "sql is required",
		}
	}
	result, err := c.api.QuerySQL(ctx, apitypes.DataProxyQueryRequest{
		ClusterID: target.ClusterID,
		SQL:       sql,
		Timeout:   query.Timeout,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) SchemaSQL(ctx context.Context, target sqlTarget, query SchemaQuery) (apitypes.DataProxySchemaResult, error) {
	result, err := c.api.SchemaSQL(ctx, apitypes.DataProxySchemaRequest{
		ClusterID: target.ClusterID,
		Tables:    append([]string(nil), query.Tables...),
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) TopSQLSummary(ctx context.Context, target resolvedClusterTarget, query TopSQLSummaryQuery) ([]apitypes.CloudTopSQL, error) {
	result, err := c.api.GetTopSQLSummary(ctx, apitypes.CloudTopSQLSummaryRequest{
		Target:    target.topSQLTarget(),
		Component: query.Component,
		Instance:  query.Instance,
		Start:     query.Start,
		End:       query.End,
		Top:       query.Top,
		Window:    query.Window,
		GroupBy:   query.GroupBy,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) TopSlowQueries(ctx context.Context, target resolvedClusterTarget, query TopSlowQueriesQuery) ([]apitypes.CloudTopSlowQuery, error) {
	result, err := c.api.GetTopSlowQueries(ctx, apitypes.CloudTopSlowQueriesRequest{
		Target:  target.topSQLTarget(),
		Start:   query.Start,
		Hours:   query.Hours,
		OrderBy: query.OrderBy,
		Limit:   query.Limit,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) SlowQuerySamples(ctx context.Context, target resolvedClusterTarget, query SlowQuerySamplesQuery) ([]apitypes.CloudSlowQueryListEntry, error) {
	if target.Identity.Platform == TargetPlatformTiUPCluster {
		resolved := buildResolvedTargetFromCloud(target)
		startTime, endTime, err := parseCollectedSlowQueryRange(query.Start, query.End)
		if err != nil {
			return nil, err
		}
		itemID, err := c.client.resolveCatalogItemID(ctx, resolved, catalogIntentSlowQueries, startTime, endTime)
		if err != nil {
			return nil, err
		}
		result, err := c.api.ListCollectedSlowQueries(ctx, apitypes.CollectedSlowQueryListRequest{
			Context:   target.Metrics.Context,
			ItemID:    itemID,
			StartTime: startTime,
			EndTime:   endTime,
			Digest:    query.Digest,
			OrderBy:   query.OrderBy,
			Limit:     query.Limit,
			Desc:      query.Desc,
		})
		return result, mapAPIError(err)
	}
	result, err := c.api.ListSlowQueries(ctx, apitypes.CloudSlowQueryListRequest{
		Target:  target.topSQLTarget(),
		Digest:  query.Digest,
		Start:   query.Start,
		End:     query.End,
		OrderBy: query.OrderBy,
		Limit:   query.Limit,
		Desc:    query.Desc,
		Fields:  append([]string(nil), query.Fields...),
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) SlowQueryDetail(ctx context.Context, target resolvedClusterTarget, query SlowQueryDetailQuery) (map[string]any, error) {
	if target.Identity.Platform == TargetPlatformTiUPCluster {
		resolved := buildResolvedTargetFromCloud(target)
		slowQueryID := strings.TrimSpace(query.ID)
		var timestamp int64
		if slowQueryID == "" {
			parsed, err := parseCollectedSlowQueryTimestamp("timestamp", query.Timestamp)
			if err != nil {
				return nil, err
			}
			timestamp = parsed
		}
		var startTime, endTime int64
		switch {
		case strings.TrimSpace(query.Start) != "" || strings.TrimSpace(query.End) != "":
			parsedStart, parsedEnd, err := parseCollectedSlowQueryRange(query.Start, query.End)
			if err != nil {
				return nil, err
			}
			startTime, endTime = parsedStart, parsedEnd
		case strings.TrimSpace(query.Timestamp) != "":
			parsed, err := parseCollectedSlowQueryTimestamp("timestamp", query.Timestamp)
			if err != nil {
				return nil, err
			}
			timestamp = parsed
			startTime, endTime = collectedSlowQueryPointWindow(timestamp)
		default:
			return nil, &Error{Class: ErrInvalidRequest, Message: "start and end are required when slow query item cannot be resolved from explicit input"}
		}
		itemID, err := c.client.resolveCatalogItemID(ctx, resolved, catalogIntentSlowQueries, startTime, endTime)
		if err != nil {
			return nil, err
		}
		if slowQueryID == "" {
			nextID, err := c.lookupCollectedSlowQueryID(ctx, target, itemID, query, timestamp)
			if err != nil {
				return nil, err
			}
			slowQueryID = nextID
		}
		result, err := c.api.GetCollectedSlowQueryDetail(ctx, apitypes.CollectedSlowQueryDetailRequest{
			Context:     target.Metrics.Context,
			ItemID:      itemID,
			SlowQueryID: slowQueryID,
		})
		if err != nil {
			return nil, mapAPIError(err)
		}
		if result == nil {
			result = map[string]any{}
		} else {
			result = cloneAnyMap(result)
		}
		if _, ok := result["id"]; !ok {
			result["id"] = slowQueryID
		}
		if _, ok := result["itemID"]; !ok {
			result["itemID"] = itemID
		}
		return result, nil
	}
	result, err := c.api.GetSlowQueryDetail(ctx, apitypes.CloudSlowQueryDetailRequest{
		Target:       target.topSQLTarget(),
		Digest:       query.Digest,
		ConnectionID: query.ConnectionID,
		Timestamp:    query.Timestamp,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) lookupCollectedSlowQueryID(ctx context.Context, target resolvedClusterTarget, itemID string, query SlowQueryDetailQuery, timestamp int64) (string, error) {
	if strings.TrimSpace(query.ConnectionID) == "" {
		return "", &Error{Class: ErrInvalidRequest, Message: "connection id is required when slow query id is not provided"}
	}
	startTime, endTime := collectedSlowQueryPointWindow(timestamp)
	samples, err := c.api.ListCollectedSlowQueries(ctx, apitypes.CollectedSlowQueryListRequest{
		Context:   target.Metrics.Context,
		ItemID:    itemID,
		StartTime: startTime,
		EndTime:   endTime,
		Digest:    query.Digest,
		OrderBy:   "timestamp",
		Limit:     100,
	})
	if err != nil {
		return "", mapAPIError(err)
	}
	for _, sample := range samples {
		if !matchesCollectedSlowQuerySample(sample, query, timestamp) {
			continue
		}
		if trimmed := strings.TrimSpace(sample.ID); trimmed != "" {
			return trimmed, nil
		}
	}
	return "", &Error{Class: ErrNotFound, Message: "slow query sample was not found in collected data"}
}
func matchesCollectedSlowQuerySample(sample apitypes.CloudSlowQueryListEntry, query SlowQueryDetailQuery, timestamp int64) bool {
	if strings.TrimSpace(query.Digest) != "" && !strings.EqualFold(strings.TrimSpace(sample.Digest), strings.TrimSpace(query.Digest)) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(sample.ConnectionID), strings.TrimSpace(query.ConnectionID)) {
		return false
	}
	sampleTimestamp, err := parseCollectedSlowQueryTimestamp("sample timestamp", sample.Timestamp)
	if err != nil {
		return strings.TrimSpace(sample.Timestamp) == strings.TrimSpace(query.Timestamp)
	}
	return sampleTimestamp == timestamp
}
func parseCollectedSlowQueryRange(start, end string) (int64, int64, error) {
	startTime, err := parseCollectedSlowQueryTimestamp("start", start)
	if err != nil {
		return 0, 0, err
	}
	endTime, err := parseCollectedSlowQueryTimestamp("end", end)
	if err != nil {
		return 0, 0, err
	}
	if endTime < startTime {
		return 0, 0, &Error{Class: ErrInvalidRequest, Message: "end must be greater than or equal to start"}
	}
	return startTime, endTime, nil
}
func parseCollectedSlowQueryTimestamp(label, value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, &Error{Class: ErrInvalidRequest, Message: fmt.Sprintf("%s is required for tiup slow query operations", label)}
	}
	if numeric, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		switch {
		case numeric >= 1_000_000_000_000:
			return numeric / 1000, nil
		default:
			return numeric, nil
		}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed.Unix(), nil
		}
	}
	return 0, &Error{Class: ErrInvalidRequest, Message: fmt.Sprintf("%s must be a unix timestamp or RFC3339 time", label)}
}
func collectedSlowQueryPointWindow(timestamp int64) (int64, int64) {
	if timestamp <= 1 {
		return timestamp, timestamp + 1
	}
	return timestamp - 1, timestamp + 1
}
func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
func (c *clinicServiceClient) SQLStatements(ctx context.Context, target sqlTarget, query SQLStatementsQuery) (apitypes.DataProxyQueryResult, error) {
	if strings.TrimSpace(query.Start) == "" || strings.TrimSpace(query.End) == "" {
		return apitypes.DataProxyQueryResult{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: dataProxyQueryEndpoint,
			Message:  "start and end are required for sql statements queries",
		}
	}
	sql := strings.TrimSpace(query.SQL)
	if sql == "" {
		return apitypes.DataProxyQueryResult{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: dataProxyQueryEndpoint,
			Message:  "sql is required for sql statements queries",
		}
	}
	sql = fmt.Sprintf("/* sql_statements start=%s end=%s */ %s", strings.TrimSpace(query.Start), strings.TrimSpace(query.End), sql)
	result, err := c.api.QuerySQL(ctx, apitypes.DataProxyQueryRequest{
		ClusterID: target.ClusterID,
		SQL:       sql,
		Timeout:   query.Timeout,
	})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) SlowQueryRecords(ctx context.Context, requestContext apitypes.RequestContext, itemID string, query SlowQueryRecordsQuery) (apitypes.SlowQueryRecordsResult, error) {
	result, err := c.api.SlowQueryRecords(ctx, apitypes.SlowQueryRequest{
		Context:   requestContext,
		ItemID:    itemID,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
		OrderBy:   query.OrderBy,
		Desc:      query.Desc,
		Limit:     query.Limit,
	})
	if err != nil {
		return apitypes.SlowQueryRecordsResult{}, mapAPIError(err)
	}
	return result, nil
}
