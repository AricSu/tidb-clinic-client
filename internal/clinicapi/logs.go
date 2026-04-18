package clinicapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func (c *lokiClient) QueryRange(ctx context.Context, req LokiQueryRangeRequest) (LokiQueryResult, error) {
	if c == nil || c.transport == nil {
		return LokiQueryResult{}, &Error{Class: ErrBackend, Message: "loki client is nil"}
	}
	if err := validateClusterIDOnly(lokiEndpointPrefix, req.ClusterID); err != nil {
		return LokiQueryResult{}, err
	}
	if strings.TrimSpace(req.Query) == "" {
		return LokiQueryResult{}, &Error{Class: ErrInvalidRequest, Endpoint: lokiQueryRangeEndpoint(req.ClusterID), Message: "query is required"}
	}
	if req.Start <= 0 || req.End <= 0 || req.End < req.Start {
		return LokiQueryResult{}, &Error{Class: ErrInvalidRequest, Endpoint: lokiQueryRangeEndpoint(req.ClusterID), Message: "valid start/end range is required"}
	}
	route, err := routeFromClusterIDOnly(lokiEndpointPrefix, req.ClusterID)
	if err != nil {
		return LokiQueryResult{}, err
	}
	query := url.Values{}
	query.Set("query", strings.TrimSpace(req.Query))
	query.Set("start", strconv.FormatInt(req.Start, 10))
	query.Set("end", strconv.FormatInt(req.End, 10))
	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}
	if direction := strings.TrimSpace(req.Direction); direction != "" {
		query.Set("direction", direction)
	}
	return doLokiQuery(ctx, c.transport, lokiQueryRangeEndpoint(req.ClusterID), query, route)
}
func (c *lokiClient) Labels(ctx context.Context, req LokiLabelsRequest) (LokiLabelsResult, error) {
	return c.getLabelList(ctx, lokiLabelsEndpoint(req.ClusterID), req.ClusterID, req.Start, req.End)
}
func (c *lokiClient) LabelValues(ctx context.Context, req LokiLabelValuesRequest) (LokiLabelsResult, error) {
	if strings.TrimSpace(req.LabelName) == "" {
		return LokiLabelsResult{}, &Error{Class: ErrInvalidRequest, Endpoint: lokiEndpointPrefix, Message: "label name is required"}
	}
	return c.getLabelList(ctx, lokiLabelValuesEndpoint(req.ClusterID, req.LabelName), req.ClusterID, req.Start, req.End)
}
func (c *lokiClient) getLabelList(ctx context.Context, endpoint, clusterID string, start, end int64) (LokiLabelsResult, error) {
	if c == nil || c.transport == nil {
		return LokiLabelsResult{}, &Error{Class: ErrBackend, Message: "loki client is nil"}
	}
	if err := validateClusterIDOnly(endpoint, clusterID); err != nil {
		return LokiLabelsResult{}, err
	}
	if (start > 0 || end > 0) && (start <= 0 || end <= 0 || end < start) {
		return LokiLabelsResult{}, &Error{Class: ErrInvalidRequest, Endpoint: endpoint, Message: "valid start/end range is required"}
	}
	route, err := routeFromClusterIDOnly(endpoint, clusterID)
	if err != nil {
		return LokiLabelsResult{}, err
	}
	query := url.Values{}
	if start > 0 {
		query.Set("start", strconv.FormatInt(start, 10))
		query.Set("end", strconv.FormatInt(end, 10))
	}
	var resp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := c.transport.getJSON(ctx, endpoint, query, nil, route.trace, &resp); err != nil {
		return LokiLabelsResult{}, err
	}
	if status := strings.ToLower(strings.TrimSpace(resp.Status)); status != "" && status != "success" {
		return LokiLabelsResult{}, &Error{Class: ErrBackend, Endpoint: endpoint, Message: fmt.Sprintf("clinic loki returned status=%s", status)}
	}
	return LokiLabelsResult{Status: strings.TrimSpace(resp.Status), Values: append([]string(nil), resp.Data...)}, nil
}
func doLokiQuery(ctx context.Context, transport *transport, endpoint string, query url.Values, route requestRoute) (LokiQueryResult, error) {
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string           `json:"resultType"`
			Result     []map[string]any `json:"result"`
		} `json:"data"`
	}
	if err := transport.getJSON(ctx, endpoint, query, nil, route.trace, &resp); err != nil {
		return LokiQueryResult{}, err
	}
	if status := strings.ToLower(strings.TrimSpace(resp.Status)); status != "" && status != "success" {
		return LokiQueryResult{}, &Error{Class: ErrBackend, Endpoint: endpoint, Message: fmt.Sprintf("clinic loki returned status=%s", status)}
	}
	out := LokiQueryResult{
		Status:     strings.TrimSpace(resp.Status),
		ResultType: strings.TrimSpace(resp.Data.ResultType),
		RawResult:  make([]map[string]any, 0, len(resp.Data.Result)),
		Streams:    make([]LokiStream, 0, len(resp.Data.Result)),
	}
	for _, item := range resp.Data.Result {
		out.RawResult = append(out.RawResult, cloneAnyMap(item))
		stream := LokiStream{
			Labels: cloneStringMap(stringMap(item["stream"])),
		}
		for _, pair := range asNestedStringPairs(item["values"]) {
			if len(pair) != 2 {
				continue
			}
			stream.Values = append(stream.Values, LokiLogValue{Timestamp: pair[0], Line: pair[1]})
		}
		out.Streams = append(out.Streams, stream)
	}
	return out, nil
}
func stringMap(v any) map[string]string {
	switch x := v.(type) {
	case map[string]string:
		return x
	case map[string]any:
		out := make(map[string]string, len(x))
		for k, raw := range x {
			out[k] = asTrimmedString(raw)
		}
		return out
	default:
		return nil
	}
}
func asNestedStringPairs(v any) [][]string {
	rawPairs, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([][]string, 0, len(rawPairs))
	for _, raw := range rawPairs {
		pairRaw, ok := raw.([]any)
		if !ok {
			continue
		}
		pair := make([]string, 0, len(pairRaw))
		for _, value := range pairRaw {
			pair = append(pair, asTrimmedString(value))
		}
		out = append(out, pair)
	}
	return out
}
