package clinicapi

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// Query returns slow query records for the selected uploaded item.
func (c *SlowQueryClient) Query(ctx context.Context, req SlowQueryRequest) (SlowQueryResult, error) {
	if c == nil || c.transport == nil {
		return SlowQueryResult{}, &Error{Class: ErrBackend, Message: "slowquery client is nil"}
	}
	route, err := routeFromItemContext(slowQueriesEndpoint, req.Context, req.ItemID)
	if err != nil {
		return SlowQueryResult{}, err
	}
	if req.StartTime <= 0 || req.EndTime <= 0 || req.EndTime < req.StartTime {
		return SlowQueryResult{}, &Error{Class: ErrInvalidRequest, Endpoint: slowQueriesEndpoint, Message: "valid start/end time range is required"}
	}
	query := url.Values{}
	query.Set("itemID", strings.TrimSpace(req.ItemID))
	query.Set("startTime", strconv.FormatInt(req.StartTime, 10))
	query.Set("endTime", strconv.FormatInt(req.EndTime, 10))
	if strings.TrimSpace(req.OrderBy) != "" {
		query.Set("orderBy", strings.TrimSpace(req.OrderBy))
	}
	if req.Desc {
		query.Set("desc", "true")
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	var resp struct {
		Total       int `json:"total"`
		SlowQueries []struct {
			Digest     string   `json:"digest"`
			Query      string   `json:"query"`
			QueryTime  float64  `json:"queryTime"`
			ExecCount  int64    `json:"execCount"`
			User       string   `json:"user"`
			DB         string   `json:"db"`
			TableNames []string `json:"tableNames"`
			IndexNames []string `json:"indexNames"`
			SourceRef  string   `json:"sourceRef"`
		} `json:"slowQueries"`
	}
	if err := c.transport.getJSON(ctx, slowQueriesEndpoint, query, route.headers, route.trace, &resp); err != nil {
		return SlowQueryResult{}, err
	}
	out := SlowQueryResult{Total: resp.Total, Records: make([]SlowQueryRecord, 0, len(resp.SlowQueries))}
	for _, row := range resp.SlowQueries {
		out.Records = append(out.Records, SlowQueryRecord{
			Digest:     strings.TrimSpace(row.Digest),
			SQLText:    row.Query,
			QueryTime:  row.QueryTime,
			ExecCount:  row.ExecCount,
			User:       row.User,
			DB:         row.DB,
			TableNames: append([]string(nil), row.TableNames...),
			IndexNames: append([]string(nil), row.IndexNames...),
			SourceRef:  row.SourceRef,
		})
	}
	return out, nil
}

// Search returns log search results for the selected uploaded item.
func (c *LogClient) Search(ctx context.Context, req LogSearchRequest) (LogSearchResult, error) {
	if c == nil || c.transport == nil {
		return LogSearchResult{}, &Error{Class: ErrBackend, Message: "log client is nil"}
	}
	route, err := routeFromItemContext(logsEndpoint, req.Context, req.ItemID)
	if err != nil {
		return LogSearchResult{}, err
	}
	query := url.Values{}
	query.Set("itemID", strings.TrimSpace(req.ItemID))
	if req.StartTime > 0 {
		query.Set("startTime", strconv.FormatInt(req.StartTime, 10))
	}
	if req.EndTime > 0 {
		query.Set("endTime", strconv.FormatInt(req.EndTime, 10))
	}
	if strings.TrimSpace(req.Pattern) != "" {
		query.Set("pattern", strings.TrimSpace(req.Pattern))
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	var resp struct {
		Total int `json:"total"`
		Logs  []struct {
			Timestamp int64  `json:"timestamp"`
			Component string `json:"component"`
			Level     string `json:"level"`
			Message   string `json:"message"`
			SourceRef string `json:"sourceRef"`
		} `json:"logs"`
	}
	if err := c.transport.getJSON(ctx, logsEndpoint, query, route.headers, route.trace, &resp); err != nil {
		return LogSearchResult{}, err
	}
	out := LogSearchResult{Total: resp.Total, Records: make([]LogRecord, 0, len(resp.Logs))}
	for _, row := range resp.Logs {
		out.Records = append(out.Records, LogRecord{
			Timestamp: row.Timestamp,
			Component: row.Component,
			Level:     row.Level,
			Message:   row.Message,
			SourceRef: row.SourceRef,
		})
	}
	return out, nil
}

// Get returns a config snapshot for the selected uploaded item.
func (c *ConfigClient) Get(ctx context.Context, req ConfigRequest) (ConfigSnapshot, error) {
	if c == nil || c.transport == nil {
		return ConfigSnapshot{}, &Error{Class: ErrBackend, Message: "config client is nil"}
	}
	route, err := routeFromItemContext(configEndpoint, req.Context, req.ItemID)
	if err != nil {
		return ConfigSnapshot{}, err
	}
	query := url.Values{}
	query.Set("itemID", strings.TrimSpace(req.ItemID))
	var resp struct {
		Configs []struct {
			Component string `json:"component"`
			Key       string `json:"key"`
			Value     string `json:"value"`
			SourceRef string `json:"sourceRef"`
		} `json:"configs"`
	}
	if err := c.transport.getJSON(ctx, configEndpoint, query, route.headers, route.trace, &resp); err != nil {
		return ConfigSnapshot{}, err
	}
	out := ConfigSnapshot{Entries: make([]ConfigEntry, 0, len(resp.Configs))}
	for _, row := range resp.Configs {
		out.Entries = append(out.Entries, ConfigEntry{
			Component: row.Component,
			Key:       row.Key,
			Value:     row.Value,
			SourceRef: row.SourceRef,
		})
	}
	return out, nil
}
