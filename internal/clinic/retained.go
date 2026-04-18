package clinic

import (
	"context"
	"errors"
	clinicapi "github.com/AricSu/tidb-clinic-client/internal/clinicapi"
	"sort"
	"strings"
)

type catalogIntent string

const (
	catalogIntentCollectedData catalogIntent = "collected_data"
	catalogIntentLogs          catalogIntent = "logs"
	catalogIntentSlowQueries   catalogIntent = "slow_queries"
)

func (c *Client) resolveCatalogItemID(ctx context.Context, target resolvedTarget, intent catalogIntent, start, end int64) (string, error) {
	if c == nil || c.clinic == nil {
		return "", &Error{Class: ErrBackend, Message: "clinic service is nil"}
	}
	requestContext, ok := target.requestContext()
	if !ok {
		return "", &Error{Class: ErrInvalidRequest, Message: "target request context is missing"}
	}
	items, err := c.clinic.ListCatalogData(ctx, requestContext)
	if err != nil {
		return "", err
	}
	item, err := selectCatalogItem(intent, items, start, end)
	if err != nil {
		return "", err
	}
	if target.Platform == TargetPlatformTiUPCluster {
		if err := c.clinic.EnsureCatalogDataReadable(ctx, requestContext, item, catalogDataTypeForIntent(intent)); err != nil {
			return "", err
		}
	}
	return item.ItemID, nil
}
func (c *clinicServiceClient) ListCatalogData(ctx context.Context, requestContext clinicapi.RequestContext) ([]clinicapi.ClinicDataItem, error) {
	result, err := c.api.ListCatalogData(ctx, clinicapi.ListClusterDataRequest{Context: requestContext})
	return result, mapAPIError(err)
}
func (c *clinicServiceClient) EnsureCatalogDataReadable(ctx context.Context, requestContext clinicapi.RequestContext, item clinicapi.ClinicDataItem, dataType clinicapi.CatalogDataType) error {
	return mapAPIError(c.api.EnsureCatalogDataReadable(ctx, clinicapi.EnsureCatalogDataReadableRequest{
		Context:  requestContext,
		Item:     item,
		DataType: dataType,
	}))
}
func catalogDataTypeForIntent(intent catalogIntent) clinicapi.CatalogDataType {
	switch intent {
	case catalogIntentLogs:
		return clinicapi.CatalogDataTypeLogs
	case catalogIntentSlowQueries:
		return clinicapi.CatalogDataTypeLogs
	case catalogIntentCollectedData:
		return clinicapi.CatalogDataTypeCollectedDownload
	default:
		return clinicapi.CatalogDataTypeRetained
	}
}
func selectCatalogItem(intent catalogIntent, items []clinicapi.ClinicDataItem, start, end int64) (clinicapi.ClinicDataItem, error) {
	type candidate struct {
		item        clinicapi.ClinicDataItem
		overlap     int64
		hasSlowLogs bool
	}
	candidates := make([]candidate, 0, len(items))
	for _, item := range items {
		if !eligibleCatalogItem(intent, item) {
			continue
		}
		candidates = append(candidates, candidate{
			item:        item,
			overlap:     rangeOverlapSeconds(start, end, item.StartTime, item.EndTime),
			hasSlowLogs: hasCollector(item.Collectors, "log.slow"),
		})
	}
	if len(candidates) == 0 {
		return clinicapi.ClinicDataItem{}, errors.New("no suitable catalog item found for " + string(intent))
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		switch intent {
		case catalogIntentCollectedData:
			if left.overlap != right.overlap {
				return left.overlap > right.overlap
			}
			if left.item.EndTime != right.item.EndTime {
				return left.item.EndTime > right.item.EndTime
			}
			if left.item.StartTime != right.item.StartTime {
				return left.item.StartTime > right.item.StartTime
			}
		case catalogIntentLogs, catalogIntentSlowQueries:
			if intent == catalogIntentSlowQueries && left.hasSlowLogs != right.hasSlowLogs {
				return left.hasSlowLogs
			}
			if left.overlap != right.overlap {
				return left.overlap > right.overlap
			}
			if left.item.EndTime != right.item.EndTime {
				return left.item.EndTime > right.item.EndTime
			}
			if left.item.StartTime != right.item.StartTime {
				return left.item.StartTime > right.item.StartTime
			}
		}
		return left.item.ItemID < right.item.ItemID
	})
	return candidates[0].item, nil
}
func eligibleCatalogItem(intent catalogIntent, item clinicapi.ClinicDataItem) bool {
	switch intent {
	case catalogIntentCollectedData:
		return true
	case catalogIntentLogs, catalogIntentSlowQueries:
		return item.HaveLog
	default:
		return false
	}
}
func rangeOverlapSeconds(startA, endA, startB, endB int64) int64 {
	if endA <= 0 || endB <= 0 {
		return 0
	}
	start := maxInt64(startA, startB)
	end := minInt64(endA, endB)
	if end <= start {
		return 0
	}
	return end - start
}
func hasCollector(collectors []string, target string) bool {
	for _, collector := range collectors {
		if strings.EqualFold(strings.TrimSpace(collector), target) {
			return true
		}
	}
	return false
}
func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}
