package compiler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/AricSu/tidb-clinic-client/internal/model"
)

func TestBuildAnalyzeRequestMapsMetricSeries(t *testing.T) {
	request, err := BuildAnalyzeRequest(model.MetricsCompileQuery{Query: "up"}, model.SeriesResult{
		Series: []model.Series{
			{
				Labels: map[string]string{
					"__name__": "tidb_qps",
					"instance": "tidb-0:10080",
					"job":      "tidb",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.5"},
					{Timestamp: 1772776860, Value: "2.5"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzeRequest failed: %v", err)
	}
	if request.Scope != "line" {
		t.Fatalf("unexpected scope: %+v", request)
	}
	if len(request.Series) != 1 {
		t.Fatalf("unexpected series count: %+v", request)
	}
	if request.Series[0].MetricID != "tidb_qps" {
		t.Fatalf("unexpected metric id: %+v", request.Series[0])
	}
	if request.Series[0].EntityID != "instance=tidb-0:10080,job=tidb" {
		t.Fatalf("unexpected entity id: %+v", request.Series[0])
	}
	if request.Series[0].GroupID != "tidb_qps" {
		t.Fatalf("unexpected group id: %+v", request.Series[0])
	}
}

func TestBuildAnalyzeRequestAutoSelectsGroupScope(t *testing.T) {
	request, err := BuildAnalyzeRequest(model.MetricsCompileQuery{Query: "up"}, model.SeriesResult{
		Series: []model.Series{
			{
				Labels: map[string]string{
					"__name__": "tidb_qps",
					"instance": "tidb-0:10080",
					"job":      "tidb-cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.5"},
					{Timestamp: 1772776860, Value: "2.5"},
					{Timestamp: 1772776920, Value: "2.1"},
				},
			},
			{
				Labels: map[string]string{
					"__name__": "tidb_qps",
					"instance": "tidb-1:10080",
					"job":      "tidb-cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.7"},
					{Timestamp: 1772776860, Value: "2.1"},
					{Timestamp: 1772776920, Value: "2.0"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzeRequest failed: %v", err)
	}
	if request.Scope != "group" {
		t.Fatalf("unexpected scope: %+v", request)
	}
	if len(request.Series) != 0 {
		t.Fatalf("expected no line series in group mode: %+v", request)
	}
	if len(request.Groups) != 1 {
		t.Fatalf("unexpected group count: %+v", request)
	}
	if request.Groups[0].MetricID != "tidb_qps" || request.Groups[0].GroupID != "tidb-cluster-a" {
		t.Fatalf("unexpected group metadata: %+v", request.Groups[0])
	}
	if len(request.Groups[0].Members) != 2 {
		t.Fatalf("unexpected group members: %+v", request.Groups[0])
	}
}

func TestBuildAnalyzeRequestAutoSelectsGroupScopeWithMinorTimestampGaps(t *testing.T) {
	request, err := BuildAnalyzeRequest(model.MetricsCompileQuery{Query: "up"}, model.SeriesResult{
		Series: []model.Series{
			{
				Labels: map[string]string{
					"__name__": "tidb_qps",
					"instance": "tidb-0:10080",
					"job":      "tidb-cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.5"},
					{Timestamp: 1772776860, Value: "2.5"},
					{Timestamp: 1772776920, Value: "2.1"},
					{Timestamp: 1772776980, Value: "2.0"},
					{Timestamp: 1772777040, Value: "2.4"},
				},
			},
			{
				Labels: map[string]string{
					"__name__": "tidb_qps",
					"instance": "tidb-1:10080",
					"job":      "tidb-cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.7"},
					{Timestamp: 1772776920, Value: "2.1"},
					{Timestamp: 1772776980, Value: "2.3"},
					{Timestamp: 1772777040, Value: "2.2"},
					{Timestamp: 1772777100, Value: "2.0"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzeRequest failed: %v", err)
	}
	if request.Scope != "group" {
		t.Fatalf("unexpected scope: %+v", request)
	}
	if len(request.Groups) != 1 || len(request.Groups[0].Members) != 2 {
		t.Fatalf("unexpected grouped request: %+v", request)
	}
	for _, member := range request.Groups[0].Members {
		if len(member.Points) != 4 {
			t.Fatalf("expected trimmed common timestamps, got %+v", member.Points)
		}
	}
}

func TestBuildAnalyzeRequestFallsBackToLineWhenSeriesCannotFormComparableGroups(t *testing.T) {
	request, err := BuildAnalyzeRequest(model.MetricsCompileQuery{Query: "up"}, model.SeriesResult{
		Series: []model.Series{
			{
				Labels: map[string]string{
					"__name__": "tidb_qps",
					"instance": "tidb-0:10080",
					"job":      "tidb-cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.5"},
					{Timestamp: 1772776860, Value: "2.5"},
					{Timestamp: 1772776920, Value: "2.1"},
					{Timestamp: 1772776980, Value: "2.0"},
				},
			},
			{
				Labels: map[string]string{
					"__name__": "tidb_qps",
					"instance": "tidb-1:10080",
					"job":      "tidb-cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772777100, Value: "1.7"},
					{Timestamp: 1772777160, Value: "2.1"},
					{Timestamp: 1772777220, Value: "2.3"},
					{Timestamp: 1772777280, Value: "2.2"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzeRequest failed: %v", err)
	}
	if request.Scope != "line" {
		t.Fatalf("unexpected scope: %+v", request)
	}
	if len(request.Series) != 2 {
		t.Fatalf("expected line fallback to preserve series: %+v", request)
	}
	if len(request.Groups) != 0 {
		t.Fatalf("expected no groups after line fallback: %+v", request)
	}
}

func TestBuildAnalyzeRequestExtractsMetricIDAndFriendlyGroupIDFromExpressionQuery(t *testing.T) {
	request, err := BuildAnalyzeRequest(model.MetricsCompileQuery{
		Query: "histogram_quantile(0.999, sum(rate(tidb_server_handle_query_duration_seconds_bucket[1m])) by (le, instance))",
	}, model.SeriesResult{
		Series: []model.Series{
			{
				Labels: map[string]string{
					"instance": "db-tidb-43",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.5"},
					{Timestamp: 1772776860, Value: "2.5"},
					{Timestamp: 1772776920, Value: "2.1"},
				},
			},
			{
				Labels: map[string]string{
					"instance": "db-tidb-44",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1772776800, Value: "1.7"},
					{Timestamp: 1772776860, Value: "2.1"},
					{Timestamp: 1772776920, Value: "2.0"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildAnalyzeRequest failed: %v", err)
	}
	if request.Scope != "group" {
		t.Fatalf("unexpected scope: %+v", request)
	}
	if len(request.Groups) != 1 {
		t.Fatalf("unexpected group count: %+v", request)
	}
	if request.Groups[0].MetricID != "tidb_server_handle_query_duration_seconds_bucket" {
		t.Fatalf("unexpected metric id: %+v", request.Groups[0])
	}
	if request.Groups[0].GroupID != "instance group" {
		t.Fatalf("unexpected group id: %+v", request.Groups[0])
	}
}

func TestCompileMetricQueryRangeRunsEmbeddedCompiler(t *testing.T) {
	resultJSON, err := CompileMetricQueryRange(context.Background(), model.MetricsCompileQuery{
		Query: "cpu_usage",
	}, model.SeriesResult{
		Series: []model.Series{
			{
				Labels: map[string]string{
					"__name__": "cpu_usage",
					"instance": "host-1",
					"job":      "cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1774951200, Value: "0.20"},
					{Timestamp: 1774951260, Value: "0.22"},
					{Timestamp: 1774951320, Value: "0.21"},
					{Timestamp: 1774951380, Value: "0.24"},
					{Timestamp: 1774951440, Value: "0.95"},
					{Timestamp: 1774951500, Value: "0.92"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CompileMetricQueryRange failed: %v", err)
	}
	if !strings.Contains(string(resultJSON), "\"subject_id\":\"instance=host-1,job=cluster-a\"") {
		t.Fatalf("unexpected compiler output: %s", string(resultJSON))
	}
	var decoded map[string]any
	if err := json.Unmarshal(resultJSON, &decoded); err != nil {
		t.Fatalf("compiler output is not valid json: %v", err)
	}
}

func TestCompileMetricQueryRangeDigestsPreservesMetricIDAndSourceRef(t *testing.T) {
	digests, err := CompileMetricQueryRangeDigests(context.Background(), model.MetricsCompileQuery{
		Query:            "cpu_usage",
		MetricID:         "tidb_cpu_usage",
		LabelsOfInterest: []string{"instance"},
		SourceRef:        "clinic metric_id=tidb_cpu_usage",
	}, model.SeriesResult{
		Series: []model.Series{
			{
				Labels: map[string]string{
					"__name__": "cpu_usage",
					"instance": "host-1",
					"job":      "cluster-a",
				},
				Values: []model.SeriesPoint{
					{Timestamp: 1774951200, Value: "0.20"},
					{Timestamp: 1774951260, Value: "0.22"},
					{Timestamp: 1774951320, Value: "0.21"},
					{Timestamp: 1774951380, Value: "0.24"},
					{Timestamp: 1774951440, Value: "0.95"},
					{Timestamp: 1774951500, Value: "0.92"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CompileMetricQueryRangeDigests failed: %v", err)
	}
	if len(digests) != 1 {
		t.Fatalf("expected one digest, got=%d", len(digests))
	}
	if digests[0].MetricID != "tidb_cpu_usage" {
		t.Fatalf("expected metric id override, got=%q", digests[0].MetricID)
	}
	if len(digests[0].SourceRefs) != 1 || digests[0].SourceRefs[0] != "clinic metric_id=tidb_cpu_usage" {
		t.Fatalf("expected source ref to be preserved, got=%+v", digests[0].SourceRefs)
	}
}
