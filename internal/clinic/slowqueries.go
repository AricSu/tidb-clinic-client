package clinic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apitypes "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
)

func (c *SlowQueryClient) Query(ctx context.Context, query SlowQueryQuery) (SlowQueryResult, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return SlowQueryResult{}, &Error{Class: ErrBackend, Message: "slow query client is nil"}
	}
	target, err := c.handle.requireTarget("slow query client")
	if err != nil {
		return SlowQueryResult{}, err
	}
	requestContext, ok := target.requestContext()
	if !ok {
		return SlowQueryResult{}, &Error{Class: ErrBackend, Message: "slow query request context is missing"}
	}
	if target.Platform == TargetPlatformCloud {
		ngmTarget, err := cloudSlowQueryTarget(target)
		if err != nil {
			return SlowQueryResult{}, err
		}
		return c.handle.client.clinic.QueryCloudSlowQueries(ctx, ngmTarget, query)
	}
	items, err := c.handle.client.clinic.ListCatalogData(ctx, requestContext)
	if err != nil {
		return SlowQueryResult{}, err
	}
	item, err := selectCatalogItem(catalogIntentSlowQueries, items, query.Start, query.End)
	if err != nil {
		return SlowQueryResult{}, err
	}
	return c.handle.client.clinic.QuerySlowQueries(ctx, requestContext, item, query)
}

func (c *SlowQueryClient) Samples(ctx context.Context, query SlowQuerySamplesQuery) (SlowQuerySamplesResult, error) {
	if c == nil || c.handle == nil || c.handle.client == nil || c.handle.client.clinic == nil {
		return SlowQuerySamplesResult{}, &Error{Class: ErrBackend, Message: "slow query client is nil"}
	}
	target, err := c.handle.requireTarget("slow query client")
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	requestContext, ok := target.requestContext()
	if !ok {
		return SlowQuerySamplesResult{}, &Error{Class: ErrBackend, Message: "slow query request context is missing"}
	}
	startTime, endTime, err := parseSlowQuerySampleRange(query.Start, query.End)
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	if target.Platform == TargetPlatformCloud {
		ngmTarget, err := cloudSlowQueryTarget(target)
		if err != nil {
			return SlowQuerySamplesResult{}, err
		}
		return c.handle.client.clinic.QueryCloudSlowQuerySamples(ctx, ngmTarget, query, startTime, endTime)
	}
	items, err := c.handle.client.clinic.ListCatalogData(ctx, requestContext)
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	item, err := selectCatalogItem(catalogIntentSlowQueries, items, startTime, endTime)
	if err != nil {
		return SlowQuerySamplesResult{}, err
	}
	return c.handle.client.clinic.QuerySlowQuerySamples(ctx, requestContext, item, query, startTime, endTime)
}

func (c *clinicServiceClient) QueryCloudSlowQueries(ctx context.Context, target apitypes.CloudNGMTarget, query SlowQueryQuery) (SlowQueryResult, error) {
	request := apitypes.CloudSlowQueryRequest{
		Target:       target,
		BeginTime:    query.Start,
		EndTime:      query.End,
		OrderBy:      query.OrderBy,
		Desc:         query.Desc,
		Limit:        query.Limit,
		ShowInternal: false,
	}
	for {
		result, err := c.api.QueryCloudSlowQueries(ctx, request)
		if err == nil {
			return result, nil
		}
		if !isSlowQueryProcessingError(err) {
			return SlowQueryResult{}, mapAPIError(err)
		}
		if err := waitForSlowQueryProcessing(ctx, c.slowQueryProbeInterval()); err != nil {
			return SlowQueryResult{}, err
		}
	}
}

func (c *clinicServiceClient) QueryCloudSlowQuerySamples(ctx context.Context, target apitypes.CloudNGMTarget, query SlowQuerySamplesQuery, startTime, endTime int64) (SlowQuerySamplesResult, error) {
	request := apitypes.CloudSlowQueryRequest{
		Target:       target,
		BeginTime:    startTime,
		EndTime:      endTime,
		OrderBy:      strings.TrimSpace(query.OrderBy),
		Desc:         query.Desc,
		Digest:       strings.TrimSpace(query.Digest),
		Fields:       append([]string(nil), query.Fields...),
		Limit:        query.Limit,
		ShowInternal: false,
	}
	for {
		result, err := c.api.QueryCloudSlowQuerySamples(ctx, request)
		if err == nil {
			return result, nil
		}
		if !isSlowQueryProcessingError(err) {
			return SlowQuerySamplesResult{}, mapAPIError(err)
		}
		if err := waitForSlowQueryProcessing(ctx, c.slowQueryProbeInterval()); err != nil {
			return SlowQuerySamplesResult{}, err
		}
	}
}

func (c *clinicServiceClient) QuerySlowQueries(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem, query SlowQueryQuery) (SlowQueryResult, error) {
	if !isCloudSlowQueryContext(requestContext) {
		if err := c.EnsureCatalogDataReadable(ctx, requestContext, item, apitypes.CatalogDataTypeLogs); err != nil {
			return SlowQueryResult{}, err
		}
	}
	request := apitypes.SlowQueryRequest{
		Context:   requestContext,
		ItemID:    item.ItemID,
		StartTime: query.Start,
		EndTime:   query.End,
		OrderBy:   query.OrderBy,
		Desc:      query.Desc,
		Limit:     query.Limit,
	}
	for {
		result, err := c.api.QuerySlowQueries(ctx, request)
		if err == nil {
			return result, nil
		}
		if !isSlowQueryProcessingError(err) {
			return SlowQueryResult{}, mapAPIError(err)
		}
		if err := waitForSlowQueryProcessing(ctx, c.slowQueryProbeInterval()); err != nil {
			return SlowQueryResult{}, err
		}
	}
}

func (c *clinicServiceClient) QuerySlowQuerySamples(ctx context.Context, requestContext apitypes.RequestContext, item apitypes.ClinicDataItem, query SlowQuerySamplesQuery, startTime, endTime int64) (SlowQuerySamplesResult, error) {
	if !isCloudSlowQueryContext(requestContext) {
		if err := c.EnsureCatalogDataReadable(ctx, requestContext, item, apitypes.CatalogDataTypeLogs); err != nil {
			return SlowQuerySamplesResult{}, err
		}
	}
	request := apitypes.SlowQuerySamplesRequest{
		Context:   requestContext,
		ItemID:    item.ItemID,
		StartTime: startTime,
		EndTime:   endTime,
		Digest:    strings.TrimSpace(query.Digest),
		OrderBy:   strings.TrimSpace(query.OrderBy),
		Desc:      query.Desc,
		Limit:     query.Limit,
		Fields:    append([]string(nil), query.Fields...),
	}
	for {
		result, err := c.api.QuerySlowQuerySamples(ctx, request)
		if err == nil {
			return result, nil
		}
		if !isSlowQueryProcessingError(err) {
			return SlowQuerySamplesResult{}, mapAPIError(err)
		}
		if err := waitForSlowQueryProcessing(ctx, c.slowQueryProbeInterval()); err != nil {
			return SlowQuerySamplesResult{}, err
		}
	}
}

func isCloudSlowQueryContext(requestContext apitypes.RequestContext) bool {
	if strings.EqualFold(strings.TrimSpace(requestContext.RoutingOrgType), "cloud") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(requestContext.OrgType), "cloud")
}

func cloudSlowQueryTarget(target resolvedTarget) (apitypes.CloudNGMTarget, error) {
	if target.Cloud == nil {
		return apitypes.CloudNGMTarget{}, &Error{Class: ErrBackend, Message: "cloud slow query target is missing"}
	}
	return apitypes.CloudNGMTarget{
		Provider:   strings.TrimSpace(target.Cloud.Provider),
		Region:     strings.TrimSpace(target.Cloud.Region),
		TenantID:   strings.TrimSpace(target.Cloud.TenantID),
		ProjectID:  strings.TrimSpace(target.Cloud.ProjectID),
		ClusterID:  strings.TrimSpace(target.Cloud.ClusterID),
		DeployType: strings.TrimSpace(target.Cloud.DeployType),
	}, nil
}

func (c *clinicServiceClient) slowQueryProbeInterval() time.Duration {
	if c == nil || c.client == nil || c.client.cfg.RebuildProbeInterval <= 0 {
		return 10 * time.Second
	}
	return c.client.cfg.RebuildProbeInterval
}

func waitForSlowQueryProcessing(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return &Error{
				Class:    ErrTimeout,
				Endpoint: "slowquery",
				Message:  "waiting for slow query processing timed out",
				Cause:    ctx.Err(),
			}
		default:
			return &Error{
				Class:    ErrBackend,
				Endpoint: "slowquery",
				Message:  "waiting for slow query processing was cancelled",
				Cause:    ctx.Err(),
			}
		}
	case <-timer.C:
		return nil
	}
}

func isSlowQueryProcessingError(err error) bool {
	if err == nil {
		return false
	}
	var apiErr *apitypes.Error
	if !strings.Contains(strings.ToLower(err.Error()), "the log is processing") {
		return false
	}
	if errors.As(err, &apiErr) && apiErr != nil {
		return apiErr.Class == apitypes.ErrInvalidRequest
	}
	return true
}

func parseSlowQuerySampleRange(start, end string) (int64, int64, error) {
	startTime, err := parseSlowQueryTimestamp("start", start)
	if err != nil {
		return 0, 0, err
	}
	endTime, err := parseSlowQueryTimestamp("end", end)
	if err != nil {
		return 0, 0, err
	}
	if endTime < startTime {
		return 0, 0, &Error{Class: ErrInvalidRequest, Message: "end must be greater than or equal to start"}
	}
	return startTime, endTime, nil
}

func parseSlowQueryTimestamp(label, value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, &Error{Class: ErrInvalidRequest, Message: fmt.Sprintf("%s is required for slow query samples", label)}
	}
	if numeric, err := parseInt64(trimmed); err == nil {
		switch {
		case numeric >= 1_000_000_000_000:
			return numeric / 1000, nil
		default:
			return numeric, nil
		}
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return 0, &Error{Class: ErrInvalidRequest, Message: fmt.Sprintf("%s must be a unix timestamp or RFC3339 time", label)}
	}
	return parsed.Unix(), nil
}

func parseInt64(value string) (int64, error) {
	var sign int64 = 1
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("empty")
	}
	if trimmed[0] == '-' {
		sign = -1
		trimmed = trimmed[1:]
	}
	var out int64
	for _, ch := range trimmed {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid")
		}
		out = out*10 + int64(ch-'0')
	}
	return sign * out, nil
}
