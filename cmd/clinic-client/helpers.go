package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	clinicapi "github.com/aric/tidb-clinic-client"
)

type opItemRequestConfig struct {
	Base    cliConfig
	OrderBy string
	Pattern string
	Limit   int
	Desc    bool
}

type opCatalogIntent string

const (
	opCatalogIntentLogs        opCatalogIntent = "op logs search"
	opCatalogIntentSlowQueries opCatalogIntent = "op slowqueries query"
	opCatalogIntentConfigs     opCatalogIntent = "op configs get"
)

type cloudEventDetailConfig struct {
	Base    cliConfig
	EventID string
}

type cloudTopSQLConfig struct {
	Base      cliConfig
	Component string
	Instance  string
	Start     string
	End       string
	Top       int
	Window    string
	GroupBy   string
}

type cloudTopSlowQueriesConfig struct {
	Base    cliConfig
	Start   string
	Hours   int
	OrderBy string
	Limit   int
}

type cloudSlowQueryListConfig struct {
	Base    cliConfig
	Digest  string
	Start   string
	End     string
	OrderBy string
	Limit   int
	Desc    bool
	Fields  []string
}

type cloudSlowQueryDetailConfig struct {
	Base         cliConfig
	Digest       string
	ConnectionID string
	Timestamp    string
}

func newSDKClient(cfg cliConfig, logger *log.Logger) (*clinicapi.Client, error) {
	return clinicapi.NewClientWithConfig(clinicapi.Config{
		BaseURL:     cfg.BaseURL,
		BearerToken: cfg.APIKey,
		Timeout:     cfg.Timeout,
		Logger:      logger,
	})
}

func resolveCloudNGMTarget(
	ctx context.Context,
	cfg cliConfig,
	cloudClusterResolver func(context.Context, clinicapi.CloudClusterLookupRequest) (clinicapi.CloudCluster, error),
) (clinicapi.CloudNGMTarget, error) {
	cluster, err := resolveCloudCluster(ctx, cfg, cloudClusterResolver)
	if err != nil {
		return clinicapi.CloudNGMTarget{}, err
	}
	return cluster.CloudNGMTarget(), nil
}

func loadOPItemRequestConfig(lookup func(string) (string, bool), now func() time.Time) (opItemRequestConfig, error) {
	base, err := loadOPConfigFromEnv(lookup, now)
	if err != nil {
		return opItemRequestConfig{}, err
	}
	orderBy, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_ORDER_BY")
	pattern, _ := optionalEnv(lookup, "CLINIC_LOG_PATTERN")
	limit := 0
	if raw, ok := optionalEnv(lookup, "CLINIC_LIMIT"); ok {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return opItemRequestConfig{}, &parseEnvError{key: "CLINIC_LIMIT", message: "must be a positive integer"}
		}
		limit = parsed
	}
	desc := false
	if raw, ok := optionalEnv(lookup, "CLINIC_DESC"); ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return opItemRequestConfig{}, &parseEnvError{key: "CLINIC_DESC", message: "must be true or false"}
		}
		desc = parsed
	}
	return opItemRequestConfig{
		Base:    base,
		OrderBy: orderBy,
		Pattern: pattern,
		Limit:   limit,
		Desc:    desc,
	}, nil
}

func loadCloudEventDetailConfig(lookup func(string) (string, bool), now func() time.Time) (cloudEventDetailConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudEventDetailConfig{}, err
	}
	eventID, err := requiredEnv(lookup, "CLINIC_EVENT_ID")
	if err != nil {
		return cloudEventDetailConfig{}, err
	}
	return cloudEventDetailConfig{Base: base, EventID: eventID}, nil
}

func loadCloudTopSQLConfig(lookup func(string) (string, bool), now func() time.Time) (cloudTopSQLConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	component, err := requiredEnv(lookup, "CLINIC_CLOUD_COMPONENT")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	instance, err := requiredEnv(lookup, "CLINIC_CLOUD_INSTANCE")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	start, err := requiredEnv(lookup, "CLINIC_CLOUD_START")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	end, err := requiredEnv(lookup, "CLINIC_CLOUD_END")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	top, err := optionalPositiveIntEnv(lookup, "CLINIC_CLOUD_TOP")
	if err != nil {
		return cloudTopSQLConfig{}, err
	}
	window, _ := optionalEnv(lookup, "CLINIC_CLOUD_WINDOW")
	groupBy, _ := optionalEnv(lookup, "CLINIC_CLOUD_GROUP_BY")
	return cloudTopSQLConfig{
		Base:      base,
		Component: component,
		Instance:  instance,
		Start:     start,
		End:       end,
		Top:       top,
		Window:    window,
		GroupBy:   groupBy,
	}, nil
}

func loadCloudTopSlowQueriesConfig(lookup func(string) (string, bool), now func() time.Time) (cloudTopSlowQueriesConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudTopSlowQueriesConfig{}, err
	}
	start, err := requiredEnv(lookup, "CLINIC_CLOUD_START")
	if err != nil {
		return cloudTopSlowQueriesConfig{}, err
	}
	hours, err := optionalPositiveIntEnv(lookup, "CLINIC_CLOUD_HOURS")
	if err != nil {
		return cloudTopSlowQueriesConfig{}, err
	}
	orderBy, _ := optionalEnv(lookup, "CLINIC_CLOUD_ORDER_BY")
	limit, err := optionalPositiveIntEnv(lookup, "CLINIC_CLOUD_LIMIT")
	if err != nil {
		return cloudTopSlowQueriesConfig{}, err
	}
	return cloudTopSlowQueriesConfig{
		Base:    base,
		Start:   start,
		Hours:   hours,
		OrderBy: orderBy,
		Limit:   limit,
	}, nil
}

func loadCloudSlowQueryListConfig(lookup func(string) (string, bool), now func() time.Time) (cloudSlowQueryListConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	digest, err := requiredEnv(lookup, "CLINIC_CLOUD_DIGEST")
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	start, err := requiredEnv(lookup, "CLINIC_CLOUD_START")
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	end, err := requiredEnv(lookup, "CLINIC_CLOUD_END")
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	orderBy, _ := optionalEnv(lookup, "CLINIC_CLOUD_ORDER_BY")
	limit, err := optionalPositiveIntEnv(lookup, "CLINIC_CLOUD_LIMIT")
	if err != nil {
		return cloudSlowQueryListConfig{}, err
	}
	desc := false
	if raw, ok := optionalEnv(lookup, "CLINIC_CLOUD_DESC"); ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return cloudSlowQueryListConfig{}, &parseEnvError{key: "CLINIC_CLOUD_DESC", message: "must be true or false"}
		}
		desc = parsed
	}
	fields := defaultCloudFields()
	if raw, ok := optionalEnv(lookup, "CLINIC_CLOUD_FIELDS"); ok {
		fields = splitCSV(raw)
	}
	return cloudSlowQueryListConfig{
		Base:    base,
		Digest:  digest,
		Start:   start,
		End:     end,
		OrderBy: orderBy,
		Limit:   limit,
		Desc:    desc,
		Fields:  fields,
	}, nil
}

func loadCloudSlowQueryDetailConfig(lookup func(string) (string, bool), now func() time.Time) (cloudSlowQueryDetailConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudSlowQueryDetailConfig{}, err
	}
	digest, err := requiredEnv(lookup, "CLINIC_CLOUD_DIGEST")
	if err != nil {
		return cloudSlowQueryDetailConfig{}, err
	}
	connectionID, err := requiredEnv(lookup, "CLINIC_CLOUD_CONNECTION_ID")
	if err != nil {
		return cloudSlowQueryDetailConfig{}, err
	}
	timestamp, err := requiredEnv(lookup, "CLINIC_CLOUD_TIMESTAMP")
	if err != nil {
		return cloudSlowQueryDetailConfig{}, err
	}
	return cloudSlowQueryDetailConfig{
		Base:         base,
		Digest:       digest,
		ConnectionID: connectionID,
		Timestamp:    timestamp,
	}, nil
}

type parseEnvError struct {
	key     string
	message string
}

func (e *parseEnvError) Error() string {
	return e.key + " " + e.message
}

func optionalPositiveIntEnv(lookup func(string) (string, bool), key string) (int, error) {
	raw, ok := optionalEnv(lookup, key)
	if !ok {
		return 0, nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return 0, &parseEnvError{key: key, message: "must be a positive integer"}
	}
	return parsed, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func defaultCloudFields() []string {
	return []string{"query", "timestamp", "query_time", "memory_max", "request_count", "connection_id"}
}

func toString(v any) string {
	return strings.TrimSpace(strings.ReplaceAll(fmt.Sprint(v), "\n", " "))
}

func resolveOPCatalogItem(ctx context.Context, client *clinicapi.Client, cfg cliConfig, intent opCatalogIntent) (clinicapi.ClinicDataItem, error) {
	items, err := client.Catalog.ListClusterData(ctx, clinicapi.ListClusterDataRequest{
		Context: cfg.Context,
	})
	if err != nil {
		return clinicapi.ClinicDataItem{}, err
	}
	return selectCatalogItemForOP(intent, items, cfg.Start, cfg.End)
}

func selectCatalogItemForOP(intent opCatalogIntent, items []clinicapi.ClinicDataItem, start, end int64) (clinicapi.ClinicDataItem, error) {
	type candidate struct {
		item        clinicapi.ClinicDataItem
		overlap     int64
		hasSlowLogs bool
	}

	candidates := make([]candidate, 0, len(items))
	for _, item := range items {
		if !eligibleCatalogItemForOP(intent, item) {
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
		case opCatalogIntentConfigs:
			if left.item.EndTime != right.item.EndTime {
				return left.item.EndTime > right.item.EndTime
			}
			if left.item.StartTime != right.item.StartTime {
				return left.item.StartTime > right.item.StartTime
			}
		case opCatalogIntentSlowQueries:
			if left.hasSlowLogs != right.hasSlowLogs {
				return left.hasSlowLogs
			}
			fallthrough
		case opCatalogIntentLogs:
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

func eligibleCatalogItemForOP(intent opCatalogIntent, item clinicapi.ClinicDataItem) bool {
	switch intent {
	case opCatalogIntentLogs:
		return item.HaveLog
	case opCatalogIntentSlowQueries:
		return item.HaveLog
	case opCatalogIntentConfigs:
		return item.HaveConfig
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

func writeResolvedCatalogItem(out io.Writer, item clinicapi.ClinicDataItem) {
	fmt.Fprintf(
		out,
		"resolved_item_id=%s start=%d end=%d collectors=%s\n",
		item.ItemID,
		item.StartTime,
		item.EndTime,
		strings.Join(item.Collectors, ","),
	)
}
