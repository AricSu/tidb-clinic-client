package clinicapi

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func (c *dataProxyClient) Query(ctx context.Context, req DataProxyQueryRequest) (DataProxyQueryResult, error) {
	if c == nil || c.transport == nil {
		return DataProxyQueryResult{}, &Error{Class: ErrBackend, Message: "data proxy client is nil"}
	}
	if err := validateClusterIDOnly(dataProxyQueryPath, req.ClusterID); err != nil {
		return DataProxyQueryResult{}, err
	}
	if strings.TrimSpace(req.SQL) == "" {
		return DataProxyQueryResult{}, &Error{Class: ErrInvalidRequest, Endpoint: dataProxyQueryPath, Message: "sql is required"}
	}
	route, err := routeFromClusterIDOnly(dataProxyQueryPath, req.ClusterID)
	if err != nil {
		return DataProxyQueryResult{}, err
	}
	var resp struct {
		Columns  []string       `json:"columns"`
		Rows     [][]string     `json:"rows"`
		Metadata map[string]any `json:"metadata"`
		Error    string         `json:"error"`
	}
	body := map[string]any{
		"sql":       strings.TrimSpace(req.SQL),
		"clusterId": strings.TrimSpace(req.ClusterID),
	}
	if req.Timeout > 0 {
		body["timeout"] = req.Timeout
	}
	if err := c.transport.postJSON(ctx, dataProxyQueryPath, nil, route.trace, body, &resp); err != nil {
		return DataProxyQueryResult{}, err
	}
	if strings.TrimSpace(resp.Error) != "" {
		return DataProxyQueryResult{}, &Error{Class: ErrBackend, Endpoint: dataProxyQueryPath, Message: strings.TrimSpace(resp.Error)}
	}
	rows := resp.Rows
	if rows == nil {
		rows = make([][]string, 0)
	}
	return DataProxyQueryResult{
		Columns: append([]string(nil), resp.Columns...),
		Rows:    rows,
		Metadata: QueryMetadata{
			RowCount:      asInt64OrZero(resp.Metadata["rowCount"]),
			BytesScanned:  asInt64OrZero(resp.Metadata["bytesScanned"]),
			ExecutionTime: asTrimmedString(resp.Metadata["executionTime"]),
			QueryID:       asTrimmedString(resp.Metadata["queryId"]),
			Engine:        asTrimmedString(resp.Metadata["engine"]),
			Vendor:        asTrimmedString(resp.Metadata["vendor"]),
			Region:        asTrimmedString(resp.Metadata["region"]),
			Partial:       asBoolOrFalse(firstPresent(resp.Metadata, "partial", "isPartial")),
			Warnings:      sliceOfStrings(firstPresent(resp.Metadata, "warnings", "warningMessages")),
			Raw:           cloneAnyMap(resp.Metadata),
		},
	}, nil
}
func (c *dataProxyClient) Schema(ctx context.Context, req DataProxySchemaRequest) (DataProxySchemaResult, error) {
	if c == nil || c.transport == nil {
		return DataProxySchemaResult{}, &Error{Class: ErrBackend, Message: "data proxy client is nil"}
	}
	if err := validateClusterIDOnly(dataProxySchemaPath, req.ClusterID); err != nil {
		return DataProxySchemaResult{}, err
	}
	tables := compactNonEmptyStrings(req.Tables)
	if len(tables) == 0 {
		return DataProxySchemaResult{}, &Error{Class: ErrInvalidRequest, Endpoint: dataProxySchemaPath, Message: "at least one table is required"}
	}
	route, err := routeFromClusterIDOnly(dataProxySchemaPath, req.ClusterID)
	if err != nil {
		return DataProxySchemaResult{}, err
	}
	query := url.Values{}
	query.Set("clusterId", strings.TrimSpace(req.ClusterID))
	query.Set("tables", strings.Join(tables, ","))
	var raw any
	if err := c.transport.getJSON(ctx, dataProxySchemaPath, query, nil, route.trace, &raw); err != nil {
		return DataProxySchemaResult{}, err
	}
	items, err := normalizeSchemaPayload(raw)
	if err != nil {
		return DataProxySchemaResult{}, &Error{Class: ErrDecode, Endpoint: dataProxySchemaPath, Message: "decode data proxy schema response", Cause: err}
	}
	out := DataProxySchemaResult{Tables: make([]DataProxyTableSchema, 0, len(items))}
	for _, item := range items {
		schema := DataProxyTableSchema{
			Database:   asTrimmedString(firstPresent(item, "database", "db")),
			Table:      asTrimmedString(firstPresent(item, "table", "tableName")),
			DataSource: asTrimmedString(firstPresent(item, "data_source", "dataSource")),
			Location:   asTrimmedString(item["location"]),
			Raw:        cloneAnyMap(item),
		}
		for _, rawColumn := range sliceOfMaps(item["columns"]) {
			schema.Columns = append(schema.Columns, DataProxySchemaColumn{
				Name:    asTrimmedString(rawColumn["name"]),
				Type:    asTrimmedString(rawColumn["type"]),
				Comment: asTrimmedString(rawColumn["comment"]),
			})
		}
		for _, rawPartition := range sliceOfMaps(item["partitions"]) {
			schema.Partitions = append(schema.Partitions, DataProxySchemaPartition{
				Name: asTrimmedString(rawPartition["name"]),
				Type: asTrimmedString(rawPartition["type"]),
			})
		}
		out.Tables = append(out.Tables, schema)
	}
	return out, nil
}
func normalizeSchemaPayload(raw any) ([]map[string]any, error) {
	switch x := raw.(type) {
	case []any:
		return sliceOfMaps(x), nil
	case map[string]any:
		if tables, ok := x["tables"]; ok {
			return sliceOfMaps(tables), nil
		}
		if data, ok := x["data"]; ok {
			return sliceOfMaps(data), nil
		}
		return []map[string]any{x}, nil
	default:
		return nil, fmt.Errorf("unsupported schema payload type %T", raw)
	}
}
func sliceOfMaps(raw any) []map[string]any {
	switch x := raw.(type) {
	case []map[string]any:
		return x
	case []any:
		out := make([]map[string]any, 0, len(x))
		for _, item := range x {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}
func (c *cloudClient) GetTopSQLSummary(ctx context.Context, req CloudTopSQLSummaryRequest) ([]CloudTopSQL, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmTopSQLEndpoint, req.Target)
	if err != nil {
		return nil, err
	}
	component := strings.TrimSpace(req.Component)
	if component == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSQLEndpoint, Message: "component is required"}
	}
	instance := strings.TrimSpace(req.Instance)
	if instance == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSQLEndpoint, Message: "instance is required"}
	}
	start := strings.TrimSpace(req.Start)
	end := strings.TrimSpace(req.End)
	if start == "" || end == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSQLEndpoint, Message: "start and end are required"}
	}
	query := url.Values{}
	query.Set("start", start)
	query.Set("end", end)
	query.Set("instance", ngmInstance(component, instance, req.Target.ClusterID))
	query.Set("instance_type", component)
	query.Set("top", strconv.Itoa(defaultInt(req.Top, 5)))
	query.Set("window", defaultString(req.Window, "60s"))
	query.Set("group_by", defaultString(req.GroupBy, "query"))
	var resp struct {
		Data []struct {
			SQLDigest         string  `json:"sql_digest"`
			SQLText           string  `json:"sql_text"`
			CPUTimeMS         float64 `json:"cpu_time_ms"`
			ExecCountPerSec   float64 `json:"exec_count_per_sec"`
			DurationPerExecMS float64 `json:"duration_per_exec_ms"`
			ScanRecordsPerSec float64 `json:"scan_records_per_sec"`
			ScanIndexesPerSec float64 `json:"scan_indexes_per_sec"`
			Plans             []struct {
				PlanDigest        string  `json:"plan_digest"`
				PlanText          string  `json:"plan_text"`
				TimestampSec      []int64 `json:"timestamp_sec"`
				CPUTimeMS         []int64 `json:"cpu_time_ms"`
				ExecCountPerSec   float64 `json:"exec_count_per_sec"`
				DurationPerExecMS float64 `json:"duration_per_exec_ms"`
				ScanRecordsPerSec float64 `json:"scan_records_per_sec"`
				ScanIndexesPerSec float64 `json:"scan_indexes_per_sec"`
			} `json:"plans"`
		} `json:"data"`
	}
	if err := c.transport.getJSON(ctx, ngmTopSQLEndpoint, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	out := make([]CloudTopSQL, 0, len(resp.Data))
	for _, sql := range resp.Data {
		next := CloudTopSQL{
			SQLDigest:         strings.TrimSpace(sql.SQLDigest),
			SQLText:           strings.TrimSpace(sql.SQLText),
			CPUTimeMS:         sql.CPUTimeMS,
			ExecCountPerSec:   sql.ExecCountPerSec,
			DurationPerExecMS: sql.DurationPerExecMS,
			ScanRecordsPerSec: sql.ScanRecordsPerSec,
			ScanIndexesPerSec: sql.ScanIndexesPerSec,
			Plans:             make([]CloudTopSQLPlan, 0, len(sql.Plans)),
		}
		for _, plan := range sql.Plans {
			next.Plans = append(next.Plans, CloudTopSQLPlan{
				PlanDigest:        strings.TrimSpace(plan.PlanDigest),
				PlanText:          strings.TrimSpace(plan.PlanText),
				TimestampSec:      append([]int64(nil), plan.TimestampSec...),
				CPUTimeMS:         append([]int64(nil), plan.CPUTimeMS...),
				ExecCountPerSec:   plan.ExecCountPerSec,
				DurationPerExecMS: plan.DurationPerExecMS,
				ScanRecordsPerSec: plan.ScanRecordsPerSec,
				ScanIndexesPerSec: plan.ScanIndexesPerSec,
			})
		}
		out = append(out, next)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPUTimeMS > out[j].CPUTimeMS })
	return out, nil
}
func (c *cloudClient) GetTopSlowQueries(ctx context.Context, req CloudTopSlowQueriesRequest) ([]CloudTopSlowQuery, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmTopSlowQueriesPath, req.Target)
	if err != nil {
		return nil, err
	}
	start := strings.TrimSpace(req.Start)
	if start == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmTopSlowQueriesPath, Message: "start is required"}
	}
	query := url.Values{}
	query.Set("begin_time", start)
	query.Set("hours", strconv.Itoa(defaultInt(req.Hours, 1)))
	query.Set("order_by", defaultString(req.OrderBy, "sum_latency"))
	query.Set("limit", strconv.Itoa(defaultInt(req.Limit, 10)))
	var resp []map[string]any
	if err := c.transport.getJSON(ctx, ngmTopSlowQueriesPath, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	out := make([]CloudTopSlowQuery, 0, len(resp))
	for _, row := range resp {
		out = append(out, CloudTopSlowQuery{
			DB:            asTrimmedString(row["db"]),
			SQLDigest:     asTrimmedString(row["sql_digest"]),
			SQLText:       asTrimmedString(row["sql_text"]),
			StatementType: asTrimmedString(row["statement_type"]),
			Count:         asInt64OrZero(row["count"]),
			SumLatency:    asFloat64OrZero(row["sum_latency"]),
			MaxLatency:    asFloat64OrZero(row["max_latency"]),
			AvgLatency:    asFloat64OrZero(row["avg_latency"]),
			SumMemory:     asFloat64OrZero(row["sum_memory"]),
			MaxMemory:     asFloat64OrZero(row["max_memory"]),
			AvgMemory:     asFloat64OrZero(row["avg_memory"]),
			SumDisk:       asFloat64OrZero(row["sum_disk"]),
			MaxDisk:       asFloat64OrZero(row["max_disk"]),
			AvgDisk:       asFloat64OrZero(row["avg_disk"]),
			Detail:        cloneAnyMap(asAnyMap(row["detail"])),
		})
	}
	return out, nil
}
func (c *cloudClient) ListSlowQueries(ctx context.Context, req CloudSlowQueryListRequest) ([]CloudSlowQueryListEntry, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmSlowQueryListPath, req.Target)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Digest) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryListPath, Message: "digest is required"}
	}
	if strings.TrimSpace(req.Start) == "" || strings.TrimSpace(req.End) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryListPath, Message: "start and end are required"}
	}
	query := url.Values{}
	query.Set("begin_time", strings.TrimSpace(req.Start))
	query.Set("end_time", strings.TrimSpace(req.End))
	query.Set("digest", strings.TrimSpace(req.Digest))
	query.Set("order_by", defaultString(req.OrderBy, "query_time"))
	query.Set("limit", strconv.Itoa(defaultInt(req.Limit, 10)))
	query.Set("desc", strconv.FormatBool(true))
	query.Set("fields", strings.Join(defaultFields(req.Fields), ","))
	var resp []map[string]any
	if err := c.transport.getJSON(ctx, ngmSlowQueryListPath, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	out := make([]CloudSlowQueryListEntry, 0, len(resp))
	for _, row := range resp {
		out = append(out, CloudSlowQueryListEntry{
			Digest:       asTrimmedString(row["digest"]),
			Query:        asTrimmedString(row["query"]),
			Timestamp:    asTrimmedString(row["timestamp"]),
			QueryTime:    asFloat64OrZero(row["query_time"]),
			MemoryMax:    asFloat64OrZero(row["memory_max"]),
			RequestCount: asInt64OrZero(row["request_count"]),
			ConnectionID: asTrimmedString(firstPresent(row, "connection_id", "connect_id")),
			Raw:          cloneAnyMap(row),
		})
	}
	return out, nil
}
func (c *cloudClient) GetSlowQueryDetail(ctx context.Context, req CloudSlowQueryDetailRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	route, err := routeFromCloudNGMTarget(ngmSlowQueryDetailPath, req.Target)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Digest) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "digest is required"}
	}
	if strings.TrimSpace(req.ConnectionID) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "connection id is required"}
	}
	if strings.TrimSpace(req.Timestamp) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "timestamp is required"}
	}
	query := url.Values{}
	query.Set("digest", strings.TrimSpace(req.Digest))
	query.Set("connect_id", strings.TrimSpace(req.ConnectionID))
	query.Set("timestamp", strings.TrimSpace(req.Timestamp))
	var resp map[string]any
	if err := c.transport.getJSON(ctx, ngmSlowQueryDetailPath, query, route.headers, route.trace, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return map[string]any{}, nil
	}
	return resp, nil
}
func (c *slowQueryClient) Query(ctx context.Context, req SlowQueryRequest) (SlowQueryRecordsResult, error) {
	if c == nil || c.transport == nil {
		return SlowQueryRecordsResult{}, &Error{Class: ErrBackend, Message: "slow query client is nil"}
	}
	endpoint := collectedSlowQueriesEndpoint(req.Context.OrgID, req.Context.ClusterID)
	route, err := routeFromItemContext(endpoint, req.Context, req.ItemID)
	if err != nil {
		return SlowQueryRecordsResult{}, err
	}
	query := url.Values{}
	query.Set("itemID", strings.TrimSpace(req.ItemID))
	if req.StartTime > 0 {
		query.Set("begin_time", strconv.FormatInt(req.StartTime, 10))
	}
	if req.EndTime > 0 {
		query.Set("end_time", strconv.FormatInt(req.EndTime, 10))
	}
	if orderBy := strings.TrimSpace(req.OrderBy); orderBy != "" {
		query.Set("orderBy", orderBy)
	}
	query.Set("desc", strconv.FormatBool(req.Desc))
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	var raw any
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &raw); err != nil {
		return SlowQueryRecordsResult{}, err
	}
	return normalizeSlowQueryRecordsResult(raw, req.ItemID), nil
}
func (c *slowQueryClient) List(ctx context.Context, req CollectedSlowQueryListRequest) ([]CloudSlowQueryListEntry, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "slow query client is nil"}
	}
	endpoint := collectedSlowQueriesEndpoint(req.Context.OrgID, req.Context.ClusterID)
	route, err := routeFromItemContext(endpoint, req.Context, req.ItemID)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("itemID", strings.TrimSpace(req.ItemID))
	if req.StartTime > 0 {
		query.Set("begin_time", strconv.FormatInt(req.StartTime, 10))
	}
	if req.EndTime > 0 {
		query.Set("end_time", strconv.FormatInt(req.EndTime, 10))
	}
	if digest := strings.TrimSpace(req.Digest); digest != "" {
		query.Set("digest", digest)
	}
	if orderBy := strings.TrimSpace(req.OrderBy); orderBy != "" {
		query.Set("orderBy", orderBy)
	}
	query.Set("desc", strconv.FormatBool(req.Desc))
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	var raw any
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &raw); err != nil {
		return nil, err
	}
	return normalizeCollectedSlowQueryList(raw, req.ItemID), nil
}
func (c *slowQueryClient) GetDetail(ctx context.Context, req CollectedSlowQueryDetailRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "slow query client is nil"}
	}
	endpoint := collectedSlowQueryDetailEndpoint(req.Context.OrgID, req.Context.ClusterID, req.SlowQueryID)
	route, err := routeFromItemContext(endpoint, req.Context, req.ItemID)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("itemID", strings.TrimSpace(req.ItemID))
	var raw map[string]any
	if err := c.transport.getJSON(ctx, endpoint, query, route.headers, route.trace, &raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return map[string]any{}, nil
	}
	raw["itemID"] = strings.TrimSpace(req.ItemID)
	return raw, nil
}
func normalizeSlowQueryRecordsResult(raw any, itemID string) SlowQueryRecordsResult {
	total, records := unwrapCollection(raw)
	if total <= 0 {
		total = len(records)
	}
	out := SlowQueryRecordsResult{
		Total:   total,
		Records: make([]SlowQueryRecord, 0, len(records)),
	}
	for _, row := range records {
		execCount := asInt64OrZero(firstPresent(row, "exec_count", "execCount", "count"))
		if execCount == 0 {
			execCount = 1
		}
		out.Records = append(out.Records, SlowQueryRecord{
			Digest:     asTrimmedString(firstPresent(row, "digest", "sql_digest")),
			SQLText:    asTrimmedString(firstPresent(row, "query", "sql_text", "sqlText")),
			QueryTime:  asFloat64OrZero(firstPresent(row, "query_time", "queryTime")),
			ExecCount:  execCount,
			User:       asTrimmedString(firstPresent(row, "user", "username")),
			DB:         asTrimmedString(firstPresent(row, "db", "database")),
			TableNames: sliceOfStrings(firstPresent(row, "table_names", "tableNames")),
			IndexNames: sliceOfStrings(firstPresent(row, "index_names", "indexNames")),
			SourceRef:  strings.TrimSpace(itemID),
		})
	}
	return out
}
func normalizeCollectedSlowQueryList(raw any, itemID string) []CloudSlowQueryListEntry {
	_, rows := unwrapCollection(raw)
	out := make([]CloudSlowQueryListEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, CloudSlowQueryListEntry{
			ID:           asTrimmedString(firstPresent(row, "id", "slow_query_id")),
			ItemID:       strings.TrimSpace(itemID),
			Digest:       asTrimmedString(firstPresent(row, "digest", "sql_digest")),
			Query:        asTrimmedString(firstPresent(row, "query", "sql_text", "sqlText")),
			Timestamp:    asTrimmedString(firstPresent(row, "timestamp", "time")),
			QueryTime:    asFloat64OrZero(firstPresent(row, "query_time", "queryTime")),
			MemoryMax:    asFloat64OrZero(firstPresent(row, "memory_max", "memoryMax")),
			RequestCount: asInt64OrZero(firstPresent(row, "request_count", "requestCount")),
			ConnectionID: asTrimmedString(firstPresent(row, "connection_id", "connect_id", "connectionID")),
			Raw:          cloneAnyMap(row),
		})
	}
	return out
}
