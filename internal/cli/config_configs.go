package cli

import (
	"strconv"
	"time"
)

type retainedItemRequestConfig struct {
	Base    cliConfig
	OrderBy string
	Pattern string
	Limit   int
	Desc    bool
}

func loadRetainedItemRequestConfig(lookup func(string) (string, bool), now func() time.Time) (retainedItemRequestConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return retainedItemRequestConfig{}, err
	}
	orderBy, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_ORDER_BY")
	pattern, _ := optionalEnv(lookup, "CLINIC_LOG_PATTERN")
	limit := 0
	if raw, ok := optionalEnv(lookup, "CLINIC_LIMIT"); ok {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return retainedItemRequestConfig{}, &parseEnvError{key: "CLINIC_LIMIT", message: "must be a positive integer"}
		}
		limit = parsed
	}
	desc := false
	if raw, ok := optionalEnv(lookup, "CLINIC_DESC"); ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return retainedItemRequestConfig{}, &parseEnvError{key: "CLINIC_DESC", message: "must be true or false"}
		}
		desc = parsed
	}
	return retainedItemRequestConfig{
		Base:    base,
		OrderBy: orderBy,
		Pattern: pattern,
		Limit:   limit,
		Desc:    desc,
	}, nil
}
