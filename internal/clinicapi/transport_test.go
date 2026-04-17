package clinicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTransportRetriesTransientFailureOnce(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:      server.URL,
		BearerToken:  "token",
		Timeout:      5 * time.Second,
		RetryMax:     1,
		RetryBackoff: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("expected retry success, got err=%v", err)
	}
	if hits.Load() != 2 {
		t.Fatalf("expected exactly 2 attempts, got=%d", hits.Load())
	}
}

func TestTransportClassifiesAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err == nil {
		t.Fatalf("expected auth error")
	}
	clinicErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T err=%v", err, err)
	}
	if clinicErr.Class != ErrAuth || clinicErr.StatusCode != http.StatusUnauthorized || clinicErr.Retryable {
		t.Fatalf("unexpected error classification: %+v", clinicErr)
	}
}

func TestTransportClassifiesBadRequestAsInvalidRequestAndExtractsJSONMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "invalid query window",
			},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err == nil {
		t.Fatalf("expected invalid request error")
	}
	clinicErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T err=%v", err, err)
	}
	if clinicErr.Class != ErrInvalidRequest || clinicErr.StatusCode != http.StatusBadRequest || clinicErr.Retryable {
		t.Fatalf("unexpected error classification: %+v", clinicErr)
	}
	if !strings.Contains(clinicErr.Message, "invalid query window") {
		t.Fatalf("expected JSON error message to be extracted, got=%q", clinicErr.Message)
	}
}

func TestErrorHelpersWorkWithWrappedClinicErrors(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &Error{
		Class:     ErrRateLimit,
		Retryable: true,
		Message:   "too many requests",
	})
	if !IsRetryable(err) {
		t.Fatalf("expected wrapped clinic error to be retryable")
	}
	if got := ClassOf(err); got != ErrRateLimit {
		t.Fatalf("unexpected error class: %s", got)
	}
	if IsRetryable(nil) {
		t.Fatalf("expected nil error to be non-retryable")
	}
	if got := ClassOf(nil); got != "" {
		t.Fatalf("expected empty class for nil error, got=%q", got)
	}
}

func TestTransportClassifiesDecodeFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total":`))
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err == nil {
		t.Fatalf("expected decode error")
	}
	clinicErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T err=%v", err, err)
	}
	if clinicErr.Class != ErrDecode || clinicErr.Retryable {
		t.Fatalf("unexpected decode classification: %+v", clinicErr)
	}
}

func TestTransportHooksReceiveRetryAndSuccessLifecycle(t *testing.T) {
	var starts []RequestInfo
	var dones []RequestResult
	var retries []RequestRetry
	var failures []RequestFailure
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:      server.URL,
		BearerToken:  "token",
		Timeout:      5 * time.Second,
		RetryMax:     1,
		RetryBackoff: time.Millisecond,
		Hooks: Hooks{
			OnRequestStart: func(info RequestInfo) { starts = append(starts, info) },
			OnRequestDone:  func(result RequestResult) { dones = append(dones, result) },
			OnRetry:        func(retry RequestRetry) { retries = append(retries, retry) },
			OnError:        func(failure RequestFailure) { failures = append(failures, failure) },
		},
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("expected retry success, got err=%v", err)
	}

	if len(starts) != 2 {
		t.Fatalf("expected 2 request starts, got=%d", len(starts))
	}
	if starts[0].Endpoint != "/clinic/api/v1/orgs/org-1/clusters/cluster-9/data" || starts[0].OrgID != "org-1" || starts[0].ClusterID != "cluster-9" {
		t.Fatalf("unexpected start hook payload: %+v", starts[0])
	}
	if len(retries) != 1 || retries[0].ErrorClass != ErrTransient || !retries[0].Retryable || retries[0].StatusCode != http.StatusBadGateway {
		t.Fatalf("unexpected retry hook payload: %+v", retries)
	}
	if len(dones) != 1 || dones[0].StatusCode != http.StatusOK || dones[0].Attempt != 2 {
		t.Fatalf("unexpected done hook payload: %+v", dones)
	}
	if len(failures) != 0 {
		t.Fatalf("expected no terminal failures, got=%+v", failures)
	}
}

func TestTransportHooksReceiveTerminalFailure(t *testing.T) {
	var failures []RequestFailure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
		Hooks: Hooks{
			OnError: func(failure RequestFailure) { failures = append(failures, failure) },
		},
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err == nil {
		t.Fatalf("expected auth error")
	}
	if len(failures) != 1 {
		t.Fatalf("expected one terminal failure hook, got=%d", len(failures))
	}
	if failures[0].ErrorClass != ErrAuth || failures[0].Retryable || failures[0].StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected failure hook payload: %+v", failures[0])
	}
}

func TestTransportLogsSuccessfulRequest(t *testing.T) {
	var logBuf bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:     server.URL,
		BearerToken: "token",
		Timeout:     5 * time.Second,
		Logger:      log.New(&logBuf, "", 0),
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("ListClusterData failed: %v", err)
	}
	text := logBuf.String()
	if !strings.Contains(text, "stage=clinic_api endpoint=/clinic/api/v1/orgs/org-1/clusters/cluster-9/data status=done") {
		t.Fatalf("expected success log, got=%s", text)
	}
	if !strings.Contains(text, "org_id=org-1") || !strings.Contains(text, "cluster_id=cluster-9") {
		t.Fatalf("expected routed context in log, got=%s", text)
	}
	if !strings.Contains(text, "attempt=1") || !strings.Contains(text, "status_code=200") {
		t.Fatalf("expected attempt and status_code in log, got=%s", text)
	}
}

func TestTransportLogsRetryableFailureAndRetry(t *testing.T) {
	var logBuf bytes.Buffer
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:      server.URL,
		BearerToken:  "token",
		Timeout:      5 * time.Second,
		RetryMax:     1,
		RetryBackoff: time.Millisecond,
		Logger:       log.New(&logBuf, "", 0),
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("expected retry success, got err=%v", err)
	}
	text := logBuf.String()
	if !strings.Contains(text, "status=retry") {
		t.Fatalf("expected retry log, got=%s", text)
	}
	if !strings.Contains(text, "error_class=transient") {
		t.Fatalf("expected transient retry log, got=%s", text)
	}
	if !strings.Contains(text, "status=done") {
		t.Fatalf("expected final success log after retry, got=%s", text)
	}
}

func TestTransportLogsStartWhenVerboseRequestLogsEnabled(t *testing.T) {
	var logBuf bytes.Buffer
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":     0,
			"dataInfos": []map[string]any{},
		})
	}))
	defer server.Close()

	client, err := NewClientWithConfig(Config{
		BaseURL:            server.URL,
		BearerToken:        "token",
		Timeout:            5 * time.Second,
		Logger:             log.New(&logBuf, "", 0),
		VerboseRequestLogs: true,
	})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.catalog.ListClusterData(context.Background(), ListClusterDataRequest{
		Context: RequestContext{OrgID: "org-1", ClusterID: "cluster-9"},
	})
	if err != nil {
		t.Fatalf("ListClusterData failed: %v", err)
	}
	text := logBuf.String()
	if !strings.Contains(text, "status=start") {
		t.Fatalf("expected start log when verbose request logs enabled, got=%s", text)
	}
}
