package clinicapi

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// QueryRange executes a metrics range query against the Clinic API.
func (c *MetricsClient) QueryRange(ctx context.Context, req MetricsQueryRangeRequest) (MetricQueryRangeResult, error) {
	if c == nil || c.transport == nil {
		return MetricQueryRangeResult{}, &Error{Class: ErrBackend, Message: "metrics client is nil"}
	}
	if strings.TrimSpace(req.Context.OrgType) == "" {
		return MetricQueryRangeResult{}, &Error{Class: ErrInvalidRequest, Endpoint: metricsEndpoint, Message: "org type is required"}
	}
	if strings.TrimSpace(req.Context.OrgID) == "" {
		return MetricQueryRangeResult{}, &Error{Class: ErrInvalidRequest, Endpoint: metricsEndpoint, Message: "org id is required"}
	}
	if strings.TrimSpace(req.Context.ClusterID) == "" {
		return MetricQueryRangeResult{}, &Error{Class: ErrInvalidRequest, Endpoint: metricsEndpoint, Message: "cluster id is required"}
	}
	if strings.TrimSpace(req.Query) == "" {
		return MetricQueryRangeResult{}, &Error{Class: ErrInvalidRequest, Endpoint: metricsEndpoint, Message: "query is required"}
	}
	if req.Start <= 0 || req.End <= 0 || req.End < req.Start {
		return MetricQueryRangeResult{}, &Error{Class: ErrInvalidRequest, Endpoint: metricsEndpoint, Message: "valid start/end range is required"}
	}
	if strings.TrimSpace(req.Step) == "" {
		return MetricQueryRangeResult{}, &Error{Class: ErrInvalidRequest, Endpoint: metricsEndpoint, Message: "step is required"}
	}
	route, err := routeFromContext(metricsEndpoint, req.Context, "")
	if err != nil {
		return MetricQueryRangeResult{}, err
	}

	query := url.Values{}
	query.Set("query", strings.TrimSpace(req.Query))
	query.Set("start", strconv.FormatInt(req.Start, 10))
	query.Set("end", strconv.FormatInt(req.End, 10))
	query.Set("step", strings.TrimSpace(req.Step))

	var resp struct {
		Status    string `json:"status"`
		IsPartial bool   `json:"isPartial"`
		Data      struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]any           `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}
	if err := c.transport.getJSON(ctx, metricsEndpoint, query, route.headers, route.trace, &resp); err != nil {
		return MetricQueryRangeResult{}, err
	}
	if status := strings.ToLower(strings.TrimSpace(resp.Status)); status != "" && status != "success" {
		return MetricQueryRangeResult{}, &Error{Class: ErrBackend, Endpoint: metricsEndpoint, Message: fmt.Sprintf("clinic metrics returned status=%s", status)}
	}
	out := MetricQueryRangeResult{
		IsPartial:  resp.IsPartial,
		ResultType: strings.TrimSpace(resp.Data.ResultType),
		Series:     make([]MetricSeries, 0, len(resp.Data.Result)),
	}
	for _, series := range resp.Data.Result {
		next := MetricSeries{
			Labels: cloneStringMap(series.Metric),
			Values: make([]MetricSample, 0, len(series.Values)),
		}
		for _, pair := range series.Values {
			if len(pair) != 2 {
				continue
			}
			ts, ok := asInt64(pair[0])
			if !ok {
				continue
			}
			next.Values = append(next.Values, MetricSample{
				Timestamp: ts,
				Value:     fmt.Sprintf("%v", pair[1]),
			})
		}
		out.Series = append(out.Series, next)
	}
	return out, nil
}

// QueryRangeWithAutoSplit executes a metrics range query and recursively splits
// the time range when Clinic rejects the request due to max sample limits.
func (c *MetricsClient) QueryRangeWithAutoSplit(ctx context.Context, req MetricsQueryRangeRequest) (MetricQueryRangeResult, error) {
	result, err := c.QueryRange(ctx, req)
	if err == nil {
		return result, nil
	}
	if !isMaxSamplesError(err) {
		return MetricQueryRangeResult{}, err
	}
	stepDuration, parseErr := time.ParseDuration(strings.TrimSpace(req.Step))
	if parseErr != nil {
		return MetricQueryRangeResult{}, &Error{
			Class:    ErrInvalidRequest,
			Endpoint: metricsEndpoint,
			Message:  "step must be a valid duration for metrics auto-split",
			Cause:    parseErr,
		}
	}
	if stepDuration <= 0 {
		return MetricQueryRangeResult{}, err
	}
	if req.End <= req.Start || (req.End-req.Start) <= int64(stepDuration/time.Second) {
		return MetricQueryRangeResult{}, err
	}
	mid := req.Start + (req.End-req.Start)/2
	if mid <= req.Start || mid >= req.End {
		return MetricQueryRangeResult{}, err
	}
	leftReq := req
	leftReq.End = mid
	rightReq := req
	rightReq.Start = mid

	left, err := c.QueryRangeWithAutoSplit(ctx, leftReq)
	if err != nil {
		return MetricQueryRangeResult{}, err
	}
	right, err := c.QueryRangeWithAutoSplit(ctx, rightReq)
	if err != nil {
		return MetricQueryRangeResult{}, err
	}
	return mergeMetricRangeResults(left, right), nil
}

func isMaxSamplesError(err error) bool {
	var clinicErr *Error
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(maxSamplesErrorMessage)) {
		return false
	}
	if clinicErr == nil {
		return true
	}
	return true
}

func mergeMetricRangeResults(left, right MetricQueryRangeResult) MetricQueryRangeResult {
	out := MetricQueryRangeResult{
		IsPartial:  left.IsPartial || right.IsPartial,
		ResultType: firstNonEmpty(left.ResultType, right.ResultType),
		Series:     make([]MetricSeries, 0, len(left.Series)+len(right.Series)),
	}
	indexByKey := map[string]int{}
	appendSeries := func(series MetricSeries) {
		key := metricSeriesKey(series.Labels)
		if idx, ok := indexByKey[key]; ok {
			out.Series[idx].Values = mergeMetricSamples(out.Series[idx].Values, series.Values)
			return
		}
		indexByKey[key] = len(out.Series)
		out.Series = append(out.Series, MetricSeries{
			Labels: cloneStringMap(series.Labels),
			Values: mergeMetricSamples(nil, series.Values),
		})
	}
	for _, series := range left.Series {
		appendSeries(series)
	}
	for _, series := range right.Series {
		appendSeries(series)
	}
	return out
}

func mergeMetricSamples(existing, incoming []MetricSample) []MetricSample {
	if len(existing) == 0 && len(incoming) == 0 {
		return nil
	}
	all := append(append([]MetricSample(nil), existing...), incoming...)
	sort.Slice(all, func(i, j int) bool {
		if all[i].Timestamp == all[j].Timestamp {
			return all[i].Value < all[j].Value
		}
		return all[i].Timestamp < all[j].Timestamp
	})
	merged := make([]MetricSample, 0, len(all))
	for _, sample := range all {
		if len(merged) > 0 && merged[len(merged)-1].Timestamp == sample.Timestamp {
			merged[len(merged)-1] = sample
			continue
		}
		merged = append(merged, sample)
	}
	return merged
}

func metricSeriesKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(labels[k])
		b.WriteByte(';')
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
