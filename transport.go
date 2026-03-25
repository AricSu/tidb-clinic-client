package clinicapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

func (t *transport) doJSON(ctx context.Context, opts requestOptions, out any) error {
	if t == nil {
		return &Error{Class: ErrBackend, Message: "clinic transport is nil"}
	}
	attempts := t.retryMax + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		start := time.Now()
		t.logStart(opts, attempt+1)
		body, meta, err := t.doOnce(ctx, opts)
		if err == nil {
			if out == nil || len(bytes.TrimSpace(body)) == 0 {
				t.logDone(opts, attempt+1, start, meta)
				return nil
			}
			if err := json.Unmarshal(body, out); err != nil {
				decodeErr := &Error{
					Class:    ErrDecode,
					Endpoint: opts.path,
					Message:  "failed to decode clinic API response",
					Cause:    err,
				}
				t.logError(opts, attempt+1, start, meta, decodeErr)
				return decodeErr
			}
			t.logDone(opts, attempt+1, start, meta)
			return nil
		}
		lastErr = err
		var clinicErr *Error
		if !errors.As(err, &clinicErr) || !clinicErr.Retryable || attempt == attempts-1 {
			t.logError(opts, attempt+1, start, meta, err)
			return err
		}
		t.logRetry(opts, attempt+1, start, meta, clinicErr)
		if sleepErr := sleepWithJitter(ctx, t.retryBackoff, t.retryJitter); sleepErr != nil {
			return sleepErr
		}
	}
	return lastErr
}

func (t *transport) getJSON(ctx context.Context, endpoint string, query url.Values, headers http.Header, trace requestTrace, out any) error {
	return t.doJSON(ctx, requestOptions{
		method:  http.MethodGet,
		path:    endpoint,
		query:   query,
		headers: headers,
		trace:   trace,
	}, out)
}

func (t *transport) doOnce(ctx context.Context, opts requestOptions) ([]byte, responseMeta, error) {
	req, err := t.buildRequest(ctx, opts)
	if err != nil {
		var clinicErr *Error
		if errors.As(err, &clinicErr) {
			return nil, responseMeta{}, clinicErr
		}
		return nil, responseMeta{}, &Error{Class: ErrInvalidRequest, Endpoint: opts.path, Message: "failed to build clinic request", Cause: err}
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, responseMeta{}, classifyTransportError(opts.path, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, responseMeta{statusCode: resp.StatusCode}, &Error{Class: ErrTransient, Retryable: true, StatusCode: resp.StatusCode, Endpoint: opts.path, Message: "failed to read clinic API response", Cause: readErr}
	}
	meta := responseMeta{statusCode: resp.StatusCode, responseBytes: len(body)}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, meta, classifyHTTPError(opts.path, resp.StatusCode, body)
	}
	return body, meta, nil
}

func (t *transport) buildRequest(ctx context.Context, opts requestOptions) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	u := *t.baseURL
	u.Path = joinURLPath(t.baseURL.Path, opts.path)
	if len(opts.query) > 0 {
		u.RawQuery = opts.query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, opts.method, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if t.authProvider != nil {
		if err := t.authProvider.Apply(req); err != nil {
			return nil, &Error{
				Class:    ErrAuth,
				Endpoint: opts.path,
				Message:  "failed to apply clinic auth",
				Cause:    err,
			}
		}
	}
	if t.userAgent != "" {
		req.Header.Set("User-Agent", t.userAgent)
	}
	for k, vals := range opts.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	return req, nil
}

func joinURLPath(basePath, suffix string) string {
	if basePath == "" {
		return path.Clean("/" + strings.TrimPrefix(suffix, "/"))
	}
	return path.Clean(strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(suffix, "/"))
}

func sleepWithJitter(ctx context.Context, backoff, jitter time.Duration) error {
	if backoff <= 0 && jitter <= 0 {
		return nil
	}
	wait := backoff
	if jitter > 0 {
		wait += time.Duration(rand.Int63n(int64(jitter) + 1))
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type requestOptions struct {
	method  string
	path    string
	query   url.Values
	headers http.Header
	trace   requestTrace
}

type requestTrace struct {
	orgType   string
	orgID     string
	clusterID string
	itemID    string
}

type responseMeta struct {
	statusCode    int
	responseBytes int
}

func (rt requestTrace) logSuffix() string {
	var b strings.Builder
	appendLogField(&b, "org_type", rt.orgType)
	appendLogField(&b, "org_id", rt.orgID)
	appendLogField(&b, "cluster_id", rt.clusterID)
	appendLogField(&b, "item_id", rt.itemID)
	return b.String()
}

func (opts requestOptions) requestInfo(attempt int) RequestInfo {
	return RequestInfo{
		Endpoint:  opts.path,
		Method:    opts.method,
		Attempt:   attempt,
		OrgType:   opts.trace.orgType,
		OrgID:     opts.trace.orgID,
		ClusterID: opts.trace.clusterID,
		ItemID:    opts.trace.itemID,
	}
}

func (opts requestOptions) requestResult(attempt int, startedAt time.Time, meta responseMeta) RequestResult {
	return RequestResult{
		RequestInfo:   opts.requestInfo(attempt),
		StatusCode:    meta.statusCode,
		Duration:      time.Since(startedAt),
		ResponseBytes: meta.responseBytes,
	}
}

func classifyTransportError(endpoint string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &Error{Class: ErrTimeout, Retryable: true, Endpoint: endpoint, Message: "clinic API request timed out", Cause: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &Error{Class: ErrTimeout, Retryable: true, Endpoint: endpoint, Message: "clinic API network timeout", Cause: err}
	}
	return &Error{Class: ErrTransient, Retryable: true, Endpoint: endpoint, Message: "clinic API request failed", Cause: err}
}

func classifyHTTPError(endpoint string, statusCode int, body []byte) error {
	msg := extractHTTPErrorMessage(statusCode, body)
	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &Error{Class: ErrAuth, Retryable: false, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
	case http.StatusNotFound:
		return &Error{Class: ErrNotFound, Retryable: false, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
	case http.StatusTooManyRequests:
		return &Error{Class: ErrRateLimit, Retryable: true, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
	case http.StatusRequestTimeout:
		return &Error{Class: ErrTimeout, Retryable: true, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
	case http.StatusBadRequest:
		return &Error{Class: ErrInvalidRequest, Retryable: false, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
	}
	if statusCode >= 500 {
		return &Error{Class: ErrTransient, Retryable: true, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
	}
	if statusCode >= 400 {
		return &Error{Class: ErrInvalidRequest, Retryable: false, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
	}
	return &Error{Class: ErrBackend, Retryable: false, StatusCode: statusCode, Endpoint: endpoint, Message: msg}
}

func extractHTTPErrorMessage(statusCode int, body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return fmt.Sprintf("clinic API returned HTTP %d", statusCode)
	}

	var envelope map[string]any
	if err := json.Unmarshal(trimmed, &envelope); err == nil {
		if msg := firstErrorMessage(envelope, "message", "error", "msg", "detail", "details"); msg != "" {
			return msg
		}
	}

	msg := strings.TrimSpace(string(trimmed))
	if msg == "" {
		return fmt.Sprintf("clinic API returned HTTP %d", statusCode)
	}
	return msg
}

func firstErrorMessage(envelope map[string]any, keys ...string) string {
	for _, key := range keys {
		if msg := extractStringValue(envelope[key]); msg != "" {
			return msg
		}
	}
	return ""
}

func extractStringValue(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case map[string]any:
		return firstErrorMessage(x, "message", "error", "msg", "detail", "details")
	default:
		return ""
	}
}

func (t *transport) logStart(opts requestOptions, attempt int) {
	if t != nil && t.logger != nil {
		t.logger.Printf("stage=clinic_api endpoint=%s status=start method=%s attempt=%d%s", opts.path, opts.method, attempt, opts.trace.logSuffix())
	}
	if t != nil && t.hooks.OnRequestStart != nil {
		t.hooks.OnRequestStart(opts.requestInfo(attempt))
	}
}

func (t *transport) logDone(opts requestOptions, attempt int, startedAt time.Time, meta responseMeta) {
	if t == nil {
		return
	}
	result := opts.requestResult(attempt, startedAt, meta)
	if t.logger != nil {
		t.logger.Printf("stage=clinic_api endpoint=%s status=done method=%s attempt=%d status_code=%d duration_ms=%d response_bytes=%d%s", opts.path, opts.method, attempt, meta.statusCode, time.Since(startedAt).Milliseconds(), meta.responseBytes, opts.trace.logSuffix())
	}
	if t.hooks.OnRequestDone != nil {
		t.hooks.OnRequestDone(result)
	}
}

func (t *transport) logRetry(opts requestOptions, attempt int, startedAt time.Time, meta responseMeta, err *Error) {
	if t == nil || err == nil {
		return
	}
	result := opts.requestResult(attempt, startedAt, meta)
	if t.logger != nil {
		t.logger.Printf("stage=clinic_api endpoint=%s status=retry method=%s attempt=%d status_code=%d duration_ms=%d response_bytes=%d error_class=%s retryable=%t err=%q%s", opts.path, opts.method, attempt, meta.statusCode, time.Since(startedAt).Milliseconds(), meta.responseBytes, err.Class, err.Retryable, err.Error(), opts.trace.logSuffix())
	}
	if t.hooks.OnRetry != nil {
		t.hooks.OnRetry(RequestRetry{
			RequestResult: result,
			ErrorClass:    err.Class,
			Retryable:     err.Retryable,
			Err:           err,
		})
	}
}

func (t *transport) logError(opts requestOptions, attempt int, startedAt time.Time, meta responseMeta, err error) {
	if t == nil || err == nil {
		return
	}
	errorClass := ""
	retryable := false
	var clinicErr *Error
	if errors.As(err, &clinicErr) {
		errorClass = string(clinicErr.Class)
		retryable = clinicErr.Retryable
	}
	result := opts.requestResult(attempt, startedAt, meta)
	if t.logger != nil {
		t.logger.Printf("stage=clinic_api endpoint=%s status=error method=%s attempt=%d status_code=%d duration_ms=%d response_bytes=%d error_class=%s retryable=%t err=%q%s", opts.path, opts.method, attempt, meta.statusCode, time.Since(startedAt).Milliseconds(), meta.responseBytes, errorClass, retryable, err.Error(), opts.trace.logSuffix())
	}
	if t.hooks.OnError != nil {
		t.hooks.OnError(RequestFailure{
			RequestResult: result,
			ErrorClass:    ErrorClass(errorClass),
			Retryable:     retryable,
			Err:           err,
		})
	}
}

func appendLogField(b *strings.Builder, key, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(trimmed)
}
