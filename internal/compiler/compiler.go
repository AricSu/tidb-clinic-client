package compiler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/AricSu/tidb-clinic-client/compiler-rs/bindings/go/pkg/compilerwasm"
	"github.com/AricSu/tidb-clinic-client/internal/model"
)

const (
	scopeLine  = "line"
	scopeGroup = "group"
)

type AnalyzeRequest struct {
	Scope           string          `json:"scope"`
	Expr            string          `json:"expr,omitempty"`
	ExprDescription string          `json:"expr_description,omitempty"`
	Series          []LogicalSeries `json:"series,omitempty"`
	Groups          []GroupInput    `json:"groups,omitempty"`
}

type TimeSeriesPoint struct {
	TSSecs int64   `json:"ts_secs"`
	Value  float64 `json:"value"`
}

type LogicalSeries struct {
	MetricID string            `json:"metric_id"`
	EntityID string            `json:"entity_id"`
	GroupID  string            `json:"group_id"`
	Labels   [][2]string       `json:"labels,omitempty"`
	Points   []TimeSeriesPoint `json:"points"`
}

type GroupInput struct {
	MetricID string          `json:"metric_id"`
	GroupID  string          `json:"group_id"`
	Members  []LogicalSeries `json:"members"`
}

type compilerAnalyzeResponse struct {
	Outputs []compilerAnalyzeRecord `json:"outputs"`
}

type compilerAnalyzeRecord struct {
	Canonical compilerCanonical `json:"canonical"`
	LLM       compilerLLM       `json:"llm"`
}

type compilerCanonical struct {
	Scope     string          `json:"scope"`
	MetricID  string          `json:"metric_id"`
	SubjectID string          `json:"subject_id"`
	State     string          `json:"state"`
	Trend     string          `json:"trend"`
	TopEvents []compilerEvent `json:"top_events"`
}

type compilerLLM struct {
	Expr            string          `json:"expr,omitempty"`
	ExprDescription string          `json:"expr_description,omitempty"`
	AnalyzeResult   json.RawMessage `json:"analyze_result"`
}

type compilerEvent struct {
	Kind        string  `json:"kind"`
	Score       float64 `json:"score"`
	StartTSSecs int64   `json:"start_ts_secs"`
	EndTSSecs   int64   `json:"end_ts_secs"`
}

type seriesDraft struct {
	MetricID string
	Labels   map[string]string
	Points   []TimeSeriesPoint
}

func BuildAnalyzeRequest(query model.MetricsCompileQuery, result model.SeriesResult) (AnalyzeRequest, error) {
	drafts := make([]seriesDraft, 0, len(result.Series))
	for idx, item := range result.Series {
		converted, ok := convertSeries(query, idx, item)
		if !ok {
			continue
		}
		drafts = append(drafts, converted)
	}
	if len(drafts) == 0 {
		return AnalyzeRequest{}, fmt.Errorf("metrics compile requires at least one series with numeric points")
	}
	series := finalizeSeries(drafts)
	if groups, ok := autoGroupSeries(series); ok {
		return AnalyzeRequest{
			Scope:           scopeGroup,
			Expr:            strings.TrimSpace(query.Query),
			ExprDescription: strings.TrimSpace(query.ExprDescription),
			Groups:          groups,
		}, nil
	}
	return AnalyzeRequest{
		Scope:           scopeLine,
		Expr:            strings.TrimSpace(query.Query),
		ExprDescription: strings.TrimSpace(query.ExprDescription),
		Series:          series,
	}, nil
}

func CompileMetricQueryRange(ctx context.Context, query model.MetricsCompileQuery, result model.SeriesResult) ([]byte, error) {
	payload, err := compileMetricQueryRangePayload(ctx, query, result)
	if err != nil {
		return nil, err
	}
	return []byte(payload), nil
}

func CompileMetricQueryRangeDigests(ctx context.Context, query model.MetricsCompileQuery, result model.SeriesResult) ([]model.CompiledTimeseriesDigest, error) {
	payload, err := compileMetricQueryRangePayload(ctx, query, result)
	if err != nil {
		return nil, err
	}
	var response compilerAnalyzeResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		return nil, fmt.Errorf("decode compiler response: %w", err)
	}
	if len(response.Outputs) == 0 {
		return nil, fmt.Errorf("compiler response did not contain outputs")
	}
	return buildCompiledDigests(query, response), nil
}

func compileMetricQueryRangePayload(ctx context.Context, query model.MetricsCompileQuery, result model.SeriesResult) (string, error) {
	request, err := BuildAnalyzeRequest(query, result)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("encode compiler request: %w", err)
	}
	client, err := compilerwasm.NewEmbedded(ctx)
	if err != nil {
		return "", err
	}
	defer client.Close(ctx)
	responseJSON, err := client.AnalyzeJSON(ctx, payload)
	if err != nil {
		return "", err
	}
	return responseJSON, nil
}

func convertSeries(query model.MetricsCompileQuery, index int, item model.Series) (seriesDraft, bool) {
	points := make([]TimeSeriesPoint, 0, len(item.Values))
	for _, point := range item.Values {
		value, err := strconv.ParseFloat(strings.TrimSpace(point.Value), 64)
		if err != nil {
			continue
		}
		points = append(points, TimeSeriesPoint{
			TSSecs: point.Timestamp,
			Value:  value,
		})
	}
	if len(points) == 0 {
		return seriesDraft{}, false
	}
	labels := filterLabels(item.Labels, query.LabelsOfInterest)
	metricID := firstNonEmpty(
		query.MetricID,
		item.Labels["__name__"],
		extractMetricIDFromQuery(query.Query),
		query.Query,
		fmt.Sprintf("series_%d", index+1),
	)
	return seriesDraft{
		MetricID: metricID,
		Labels:   cloneStringMap(labels),
		Points:   points,
	}, true
}

func finalizeSeries(drafts []seriesDraft) []LogicalSeries {
	if len(drafts) == 0 {
		return nil
	}

	type bucketKey struct {
		metricID string
	}

	buckets := make(map[bucketKey][]int, len(drafts))
	order := make([]bucketKey, 0, len(drafts))
	for idx, draft := range drafts {
		key := bucketKey{metricID: draft.MetricID}
		if _, ok := buckets[key]; !ok {
			order = append(order, key)
		}
		buckets[key] = append(buckets[key], idx)
	}

	out := make([]LogicalSeries, len(drafts))
	for _, key := range order {
		indexes := buckets[key]
		common := commonLabels(drafts, indexes)
		groupID := firstNonEmpty(key.metricID, "metrics")
		if len(indexes) > 1 {
			if rendered := renderIdentity(common, ""); rendered != "" {
				groupID = rendered
			} else if fallback := fallbackGroupIdentity(drafts, indexes); fallback != "" {
				groupID = fallback
			}
		} else {
			common = nil
		}
		for orderIdx, draftIdx := range indexes {
			draft := drafts[draftIdx]
			entityLabels := excludeLabels(draft.Labels, common)
			entityID := renderIdentity(entityLabels, "")
			if entityID == "" {
				entityID = renderIdentity(draft.Labels, "")
			}
			if entityID == "" {
				entityID = key.metricID + "#" + strconv.Itoa(orderIdx)
			}
			out[draftIdx] = LogicalSeries{
				MetricID: draft.MetricID,
				EntityID: entityID,
				GroupID:  groupID,
				Labels:   sortedLabelPairs(draft.Labels),
				Points:   append([]TimeSeriesPoint(nil), draft.Points...),
			}
		}
	}
	return out
}

func fallbackGroupIdentity(drafts []seriesDraft, indexes []int) string {
	keys := sharedLabelKeys(drafts, indexes)
	switch len(keys) {
	case 0:
		return "series group"
	case 1:
		return keys[0] + " group"
	default:
		return strings.Join(keys, "/") + " group"
	}
}

func sharedLabelKeys(drafts []seriesDraft, indexes []int) []string {
	if len(indexes) == 0 {
		return nil
	}
	shared := make(map[string]struct{})
	for key := range excludeReservedLabels(drafts[indexes[0]].Labels) {
		shared[key] = struct{}{}
	}
	for _, idx := range indexes[1:] {
		next := excludeReservedLabels(drafts[idx].Labels)
		for key := range shared {
			if strings.TrimSpace(next[key]) == "" {
				delete(shared, key)
			}
		}
		if len(shared) == 0 {
			return nil
		}
	}
	keys := make([]string, 0, len(shared))
	for key := range shared {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func autoGroupSeries(series []LogicalSeries) ([]GroupInput, bool) {
	if len(series) < 2 {
		return nil, false
	}

	type bucketKey struct {
		metricID string
		groupID  string
	}

	buckets := make(map[bucketKey][]LogicalSeries)
	order := make([]bucketKey, 0, len(series))
	for _, item := range series {
		key := bucketKey{metricID: item.MetricID, groupID: item.GroupID}
		if _, ok := buckets[key]; !ok {
			order = append(order, key)
		}
		buckets[key] = append(buckets[key], item)
	}

	groups := make([]GroupInput, 0, len(order))
	covered := 0
	for _, key := range order {
		members := buckets[key]
		group, ok := buildComparableGroup(key.metricID, key.groupID, members)
		if !ok {
			return nil, false
		}
		groups = append(groups, group)
		covered += len(group.Members)
	}

	if len(groups) == 0 || covered != len(series) {
		return nil, false
	}
	return groups, true
}

func buildComparableGroup(metricID, groupID string, members []LogicalSeries) (GroupInput, bool) {
	if len(members) < 2 {
		return GroupInput{}, false
	}

	seenEntities := make(map[string]struct{}, len(members))
	minPoints := 0
	for idx, member := range members {
		if strings.TrimSpace(member.EntityID) == "" {
			return GroupInput{}, false
		}
		if _, ok := seenEntities[member.EntityID]; ok {
			return GroupInput{}, false
		}
		seenEntities[member.EntityID] = struct{}{}
		if member.MetricID != metricID || member.GroupID != groupID {
			return GroupInput{}, false
		}
		if len(member.Points) < 3 {
			return GroupInput{}, false
		}
		if idx == 0 || len(member.Points) < minPoints {
			minPoints = len(member.Points)
		}
	}

	commonTimestamps := intersectTimestamps(members)
	if len(commonTimestamps) < 3 {
		return GroupInput{}, false
	}
	if len(commonTimestamps)*3 < minPoints*2 {
		return GroupInput{}, false
	}

	trimmed := trimMembersToTimestamps(members, commonTimestamps)
	if len(trimmed) != len(members) {
		return GroupInput{}, false
	}

	return GroupInput{
		MetricID: metricID,
		GroupID:  groupID,
		Members:  trimmed,
	}, len(seenEntities) == len(trimmed)
}

func intersectTimestamps(members []LogicalSeries) []int64 {
	if len(members) == 0 {
		return nil
	}

	counts := make(map[int64]int)
	for idx, member := range members {
		seen := make(map[int64]struct{}, len(member.Points))
		for _, point := range member.Points {
			if _, ok := seen[point.TSSecs]; ok {
				continue
			}
			seen[point.TSSecs] = struct{}{}
			if idx == 0 {
				counts[point.TSSecs] = 1
				continue
			}
			if counts[point.TSSecs] == idx {
				counts[point.TSSecs]++
			}
		}
	}

	common := make([]int64, 0, len(counts))
	for ts, count := range counts {
		if count == len(members) {
			common = append(common, ts)
		}
	}
	sort.Slice(common, func(i, j int) bool { return common[i] < common[j] })
	return common
}

func trimMembersToTimestamps(members []LogicalSeries, timestamps []int64) []LogicalSeries {
	if len(timestamps) == 0 {
		return nil
	}

	target := make(map[int64]int, len(timestamps))
	for idx, ts := range timestamps {
		target[ts] = idx
	}

	trimmed := make([]LogicalSeries, 0, len(members))
	for _, member := range members {
		points := make([]TimeSeriesPoint, len(timestamps))
		filled := make([]bool, len(timestamps))
		for _, point := range member.Points {
			idx, ok := target[point.TSSecs]
			if !ok {
				continue
			}
			points[idx] = point
			filled[idx] = true
		}
		for _, ok := range filled {
			if !ok {
				return nil
			}
		}
		member.Points = points
		trimmed = append(trimmed, member)
	}
	return trimmed
}

func filterLabels(all map[string]string, allowed []string) map[string]string {
	if len(all) == 0 {
		return nil
	}
	if len(allowed) == 0 {
		return cloneStringMap(all)
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, raw := range allowed {
		if key := strings.TrimSpace(raw); key != "" {
			allowedSet[key] = struct{}{}
		}
	}
	if len(allowedSet) == 0 {
		return nil
	}
	out := make(map[string]string, len(allowedSet))
	for key, value := range all {
		label := strings.TrimSpace(key)
		if label == "" {
			continue
		}
		if _, ok := allowedSet[label]; !ok {
			continue
		}
		out[label] = strings.TrimSpace(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func sortedLabelPairs(labels map[string]string) [][2]string {
	if len(labels) == 0 {
		return nil
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([][2]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, [2]string{key, labels[key]})
	}
	return out
}

func commonLabels(drafts []seriesDraft, indexes []int) map[string]string {
	if len(indexes) == 0 {
		return nil
	}
	first := excludeReservedLabels(drafts[indexes[0]].Labels)
	if len(first) == 0 {
		return nil
	}
	out := cloneStringMap(first)
	for _, idx := range indexes[1:] {
		next := excludeReservedLabels(drafts[idx].Labels)
		for key, value := range out {
			if strings.TrimSpace(next[key]) != value {
				delete(out, key)
			}
		}
		if len(out) == 0 {
			return nil
		}
	}
	return out
}

func excludeReservedLabels(labels map[string]string) map[string]string {
	out := make(map[string]string, len(labels))
	for key, value := range labels {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || trimmedKey == "__name__" {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}
		out[trimmedKey] = trimmedValue
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func excludeLabels(labels, excluded map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}
	out := make(map[string]string, len(labels))
	for key, value := range excludeReservedLabels(labels) {
		if strings.TrimSpace(excluded[key]) == strings.TrimSpace(value) && strings.TrimSpace(value) != "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func renderIdentity(labels map[string]string, fallback string) string {
	labels = excludeReservedLabels(labels)
	if len(labels) == 0 {
		return strings.TrimSpace(fallback)
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 1 {
		return strings.TrimSpace(labels[keys[0]])
	}
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strings.TrimSpace(labels[key]))
	}
	return strings.Join(parts, ",")
}

func buildCompiledDigests(query model.MetricsCompileQuery, response compilerAnalyzeResponse) []model.CompiledTimeseriesDigest {
	sourceRef := strings.TrimSpace(query.SourceRef)
	metricIDOverride := strings.TrimSpace(query.MetricID)
	digests := make([]model.CompiledTimeseriesDigest, 0, len(response.Outputs))
	for _, output := range response.Outputs {
		scope := normalizedScope(output.Canonical.Scope)
		metricID := firstNonEmpty(metricIDOverride, output.Canonical.MetricID)
		digest := model.CompiledTimeseriesDigest{
			MetricID:  metricID,
			Scope:     scope,
			SubjectID: strings.TrimSpace(output.Canonical.SubjectID),
			Summary:   llmAnalyzeResultSummary(output.LLM.AnalyzeResult),
			State:     strings.TrimSpace(output.Canonical.State),
			Trend:     strings.TrimSpace(output.Canonical.Trend),
		}
		if sourceRef != "" {
			digest.SourceRefs = []string{sourceRef}
		}
		if problemRange := inferProblemRange(output.Canonical.TopEvents); problemRange != nil {
			digest.ProblemRange = problemRange
		}
		if len(output.Canonical.TopEvents) > 0 {
			digest.TopEvents = make([]model.CompiledTimeseriesEvent, 0, len(output.Canonical.TopEvents))
			for _, event := range output.Canonical.TopEvents {
				digest.TopEvents = append(digest.TopEvents, model.CompiledTimeseriesEvent{
					Kind:        strings.TrimSpace(event.Kind),
					Score:       event.Score,
					StartTSSecs: event.StartTSSecs,
					EndTSSecs:   event.EndTSSecs,
				})
			}
		}
		digests = append(digests, digest)
	}
	return digests
}

func llmAnalyzeResultSummary(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	var decoded struct {
		Summary struct {
			Text string `json:"text"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(raw, &decoded); err == nil {
		if text := strings.TrimSpace(decoded.Summary.Text); text != "" {
			return text
		}
	}
	return trimmed
}

func normalizedScope(values ...string) string {
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case scopeGroup:
			return scopeGroup
		case scopeLine:
			return scopeLine
		}
	}
	return ""
}

func inferProblemRange(events []compilerEvent) *model.CompiledTimeseriesProblemRange {
	if len(events) == 0 {
		return nil
	}
	start := events[0].StartTSSecs
	end := events[0].EndTSSecs
	for _, event := range events[1:] {
		if event.StartTSSecs < start {
			start = event.StartTSSecs
		}
		if event.EndTSSecs > end {
			end = event.EndTSSecs
		}
	}
	return &model.CompiledTimeseriesProblemRange{
		StartTSSecs: start,
		EndTSSecs:   end,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func extractMetricIDFromQuery(query string) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return ""
	}
	const keyword = "keyword"
	firstCandidate := ""
	for idx := 0; idx < len(trimmed); {
		r := rune(trimmed[idx])
		if !isIdentifierStart(r) {
			idx++
			continue
		}
		start := idx
		idx++
		for idx < len(trimmed) && isIdentifierPart(rune(trimmed[idx])) {
			idx++
		}
		token := trimmed[start:idx]
		lower := strings.ToLower(token)
		if promQLIdentifierClass(trimmed, idx, lower) == keyword {
			continue
		}
		if firstCandidate == "" {
			firstCandidate = token
		}
		if strings.Contains(token, "_") || strings.Contains(token, ":") {
			return token
		}
	}
	return firstCandidate
}

func promQLIdentifierClass(query string, end int, lower string) string {
	switch lower {
	case "by", "without", "on", "ignoring", "group_left", "group_right", "bool", "offset":
		return "keyword"
	}
	for end < len(query) && unicode.IsSpace(rune(query[end])) {
		end++
	}
	if end < len(query) && query[end] == '(' {
		return "keyword"
	}
	return ""
}

func isIdentifierStart(r rune) bool {
	return r == '_' || r == ':' || unicode.IsLetter(r)
}

func isIdentifierPart(r rune) bool {
	return isIdentifierStart(r) || unicode.IsDigit(r)
}
