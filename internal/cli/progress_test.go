package cli

import (
	"bytes"
	clinicapi "github.com/AricSu/tidb-clinic-client"
	"log"
	"strings"
	"testing"
	"time"
)

func TestCollectedDataProgressFailureIncludesElapsed(t *testing.T) {
	var buf bytes.Buffer
	progress := newDataDownloadProgress(log.New(&buf, "", 0))

	progress.onRequestError(clinicapi.RequestFailure{
		RequestResult: clinicapi.RequestResult{
			RequestInfo: clinicapi.RequestInfo{
				Endpoint: "/clinic/api/v1/orgs/org-1/clusters/cluster-9/download/item-1",
			},
			Duration: 1500 * time.Millisecond,
		},
	})

	if !strings.Contains(buf.String(), "bundle 下载失败（1.5s）") {
		t.Fatalf("expected failure log to include elapsed time, got=%q", buf.String())
	}
}

func TestCollectedDataProgressReadyMessageIncludesRebuildWaitElapsed(t *testing.T) {
	var buf bytes.Buffer
	progress := newDataDownloadProgress(log.New(&buf, "", 0))
	progress.rebuildTicker = progressTicker{
		stop:      make(chan struct{}),
		startedAt: time.Now().Add(-1500 * time.Millisecond),
	}

	progress.onRequestStart(clinicapi.RequestInfo{
		Endpoint: "/clinic/api/v1/orgs/org-1/clusters/cluster-9/download/item-1",
	})

	text := buf.String()
	if !strings.Contains(text, "bundle 已就绪（等待 1.5s）。") {
		t.Fatalf("expected ready log to include rebuild wait, got=%q", text)
	}
	if !strings.Contains(text, "正在下载 bundle...") {
		t.Fatalf("expected operation start log after ready message, got=%q", text)
	}
}

func TestCollectedDataProgressPrintsRebuildTriggerOnce(t *testing.T) {
	var buf bytes.Buffer
	progress := newRetainedSlowQueriesProgress(log.New(&buf, "", 0))

	progress.onRequestStart(clinicapi.RequestInfo{
		Endpoint: "/clinic/api/v1/orgs/org-1/clusters/cluster-9/rebuild",
	})
	progress.onRequestStart(clinicapi.RequestInfo{
		Endpoint: "/clinic/api/v1/orgs/org-1/clusters/cluster-9/rebuild",
	})

	text := buf.String()
	if strings.Count(text, "bundle 需要 rebuild，正在触发 rebuild...") != 1 {
		t.Fatalf("expected rebuild trigger log once, got=%q", text)
	}
}

func TestCollectedDataProgressSkipsTerminalFailureForSlowQueryProcessing(t *testing.T) {
	var buf bytes.Buffer
	progress := newRetainedSlowQueriesProgress(log.New(&buf, "", 0))
	progress.operationTicker = progressTicker{
		stop:      make(chan struct{}),
		startedAt: time.Now(),
	}

	progress.onRequestError(clinicapi.RequestFailure{
		RequestResult: clinicapi.RequestResult{
			RequestInfo: clinicapi.RequestInfo{
				Endpoint: "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries",
			},
			Duration: 500 * time.Millisecond,
		},
		Err: &clinicapi.Error{
			Class:    clinicapi.ErrInvalidRequest,
			Endpoint: "/clinic/api/v1/orgs/org-1/clusters/cluster-9/slowqueries",
			Message:  "the log is processing",
		},
	})

	if strings.Contains(buf.String(), "slow query 查询失败") {
		t.Fatalf("expected processing error to avoid terminal failure log, got=%q", buf.String())
	}
	if !strings.Contains(buf.String(), "slow query 正在 rebuilding，等待完成...") {
		t.Fatalf("expected processing error to print rebuilding state, got=%q", buf.String())
	}
	if progress.operationTicker.stop == nil {
		t.Fatalf("expected operation ticker to remain active while processing")
	}
}
