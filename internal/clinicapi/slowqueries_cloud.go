package clinicapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (c *cloudClient) QuerySlowQueries(ctx context.Context, req CloudSlowQueryRequest) (SlowQueryResult, error) {
	raw, err := c.querySlowQueriesRaw(ctx, req)
	if err != nil {
		return SlowQueryResult{}, err
	}
	return decodeSlowQueryResult(raw), nil
}

func (c *cloudClient) QuerySlowQuerySamples(ctx context.Context, req CloudSlowQueryRequest) (SlowQuerySamplesResult, error) {
	req.Fields = ensureCloudSlowQuerySampleLocatorFields(req.Fields)
	raw, err := c.querySlowQueriesRaw(ctx, req)
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	items := slowQuerySampleItems(raw)
	detailed := make([]map[string]any, 0, len(items))
	for _, item := range items {
		detail, err := c.QuerySlowQueryDetail(ctx, CloudSlowQueryDetailRequest{
			Target:       req.Target,
			ConnectionID: asTrimmedString(firstPresent(item, "connection_id", "connectionID", "connect_id", "connectID")),
			Digest:       firstNonEmptyString(asTrimmedString(firstPresent(item, "digest")), strings.TrimSpace(req.Digest)),
			Timestamp:    asSlowQueryDetailTimestamp(firstPresent(item, "timestamp", "ts")),
		})
		if err != nil {
			return SlowQuerySamplesResult{}, err
		}
		detailed = append(detailed, detail)
	}
	return decodeSlowQuerySamplesResult(detailed, ""), nil
}

func (c *cloudClient) QuerySlowQueryDetail(ctx context.Context, req CloudSlowQueryDetailRequest) (map[string]any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if strings.TrimSpace(req.ConnectionID) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "connect_id is required"}
	}
	if strings.TrimSpace(req.Digest) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "digest is required"}
	}
	if strings.TrimSpace(req.Timestamp) == "" {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryDetailPath, Message: "timestamp is required"}
	}
	route, err := routeFromCloudNGMTarget(ngmSlowQueryDetailPath, req.Target)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("connect_id", strings.TrimSpace(req.ConnectionID))
	query.Set("digest", strings.TrimSpace(req.Digest))
	query.Set("timestamp", strings.TrimSpace(req.Timestamp))
	var raw map[string]any
	if err := c.transport.getJSON(ctx, ngmSlowQueryDetailPath, query, route.headers, route.trace, &raw); err != nil {
		return nil, err
	}
	return cloneAnyMap(raw), nil
}

func (c *cloudClient) querySlowQueriesRaw(ctx context.Context, req CloudSlowQueryRequest) (any, error) {
	if c == nil || c.transport == nil {
		return nil, &Error{Class: ErrBackend, Message: "cloud client is nil"}
	}
	if req.BeginTime <= 0 || req.EndTime <= 0 || req.EndTime < req.BeginTime {
		return nil, &Error{Class: ErrInvalidRequest, Endpoint: ngmSlowQueryListPath, Message: "valid begin/end time range is required"}
	}
	route, err := routeFromCloudNGMTarget(ngmSlowQueryListPath, req.Target)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("begin_time", strconv.FormatInt(req.BeginTime, 10))
	query.Set("end_time", strconv.FormatInt(req.EndTime, 10))
	query.Set("desc", strconv.FormatBool(req.Desc))
	query.Set("digest", strings.TrimSpace(req.Digest))
	query.Set("text", strings.TrimSpace(req.Text))
	query.Set("show_internal", strconv.FormatBool(req.ShowInternal))
	if orderBy := strings.TrimSpace(req.OrderBy); orderBy != "" {
		query.Set("orderBy", orderBy)
	}
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	inputFields := req.Fields
	if len(inputFields) == 0 {
		inputFields = defaultCloudSlowQueryFields()
	}
	fields := make([]string, 0, len(inputFields))
	for _, field := range inputFields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			fields = append(fields, trimmed)
		}
	}
	if len(fields) > 0 {
		query.Set("fields", strings.Join(fields, ","))
	}
	var raw any
	if err := c.transport.getJSON(ctx, ngmSlowQueryListPath, query, route.headers, route.trace, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func ensureCloudSlowQuerySampleLocatorFields(fields []string) []string {
	out := make([]string, 0, len(fields)+2)
	seen := map[string]bool{}
	appendField := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, trimmed)
	}
	for _, field := range fields {
		appendField(field)
	}
	appendField("timestamp")
	appendField("connection_id")
	return out
}

func defaultCloudSlowQueryFields() []string {
	return []string{"query", "timestamp", "query_time", "memory_max", "request_count", "connection_id"}
}

func slowQuerySampleItems(raw any) []map[string]any {
	total, items := unwrapCollection(raw)
	switch value := raw.(type) {
	case []any:
		items = sliceOfMaps(value)
		if total == 0 {
			total = len(items)
		}
	case []map[string]any:
		items = value
	case map[string]any:
		switch {
		case value["slowQueries"] != nil:
			items = sliceOfMaps(value["slowQueries"])
		case value["items"] != nil:
			items = sliceOfMaps(value["items"])
		default:
			items = []map[string]any{value}
		}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, cloneAnyMap(item))
	}
	return out
}

func asSlowQueryDetailTimestamp(value any) string {
	switch x := value.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', -1, 64)
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case jsonNumberLike:
		return x.String()
	case nil:
		return ""
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

type jsonNumberLike interface {
	String() string
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
