package cli

import (
	"strconv"
	"strings"
	"time"
)

const (
	defaultSlowQuerySearchOrderBy = "query_time"
	defaultSlowQuerySearchLimit   = 100
)

type slowQueryConfig struct {
	Base    cliConfig
	Digest  string
	OrderBy string
	Limit   int
	Desc    bool
	Fields  []string
}

type slowQueryFlagInputs struct {
	Digest     string
	DigestSet  bool
	OrderBy    string
	OrderBySet bool
	Limit      int
	LimitSet   bool
	Desc       bool
	DescSet    bool
	Fields     string
	FieldsSet  bool
}

var activeSlowQueryFlagInputs slowQueryFlagInputs

func pushSlowQueryFlagInputs(next slowQueryFlagInputs) func() {
	previous := activeSlowQueryFlagInputs
	activeSlowQueryFlagInputs = next
	return func() {
		activeSlowQueryFlagInputs = previous
	}
}

func loadSlowQueryConfig(lookup func(string) (string, bool), now func() time.Time) (slowQueryConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return slowQueryConfig{}, err
	}
	digest, orderBy, limit, desc, fields, err := resolveSlowQueryInputs(lookup)
	if err != nil {
		return slowQueryConfig{}, err
	}
	return slowQueryConfig{
		Base:    base,
		Digest:  digest,
		OrderBy: orderBy,
		Limit:   limit,
		Desc:    desc,
		Fields:  fields,
	}, nil
}

func resolveSlowQueryInputs(lookup func(string) (string, bool)) (string, string, int, bool, []string, error) {
	digest, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_DIGEST")
	orderBy, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_ORDER_BY")
	limit, err := firstPositiveIntEnv(lookup, "CLINIC_SLOWQUERY_LIMIT", "CLINIC_LIMIT")
	if err != nil {
		return "", "", 0, false, nil, err
	}
	desc, err := firstBoolEnv(lookup, "CLINIC_SLOWQUERY_DESC", "CLINIC_DESC")
	if err != nil {
		return "", "", 0, false, nil, err
	}
	fieldsRaw, _ := optionalEnv(lookup, "CLINIC_SLOWQUERY_FIELDS")
	fields := splitCSV(fieldsRaw)
	if strings.TrimSpace(orderBy) == "" {
		orderBy = defaultSlowQuerySearchOrderBy
	}
	if limit <= 0 {
		limit = defaultSlowQuerySearchLimit
	}
	if activeSlowQueryFlagInputs.DigestSet || activeSlowQueryFlagInputs.OrderBySet || activeSlowQueryFlagInputs.LimitSet || activeSlowQueryFlagInputs.DescSet || activeSlowQueryFlagInputs.FieldsSet {
		if activeSlowQueryFlagInputs.LimitSet && activeSlowQueryFlagInputs.Limit <= 0 {
			return "", "", 0, false, nil, &parseEnvError{key: "--limit", message: "must be a positive integer"}
		}
		if activeSlowQueryFlagInputs.DigestSet {
			digest = strings.TrimSpace(activeSlowQueryFlagInputs.Digest)
		}
		if activeSlowQueryFlagInputs.OrderBySet {
			orderBy = strings.TrimSpace(activeSlowQueryFlagInputs.OrderBy)
		}
		if activeSlowQueryFlagInputs.LimitSet {
			limit = activeSlowQueryFlagInputs.Limit
		}
		if activeSlowQueryFlagInputs.DescSet {
			desc = activeSlowQueryFlagInputs.Desc
		}
		if activeSlowQueryFlagInputs.FieldsSet {
			fields = splitCSV(activeSlowQueryFlagInputs.Fields)
		}
	}
	return digest, orderBy, limit, desc, fields, nil
}

func firstPositiveIntEnv(lookup func(string) (string, bool), keys ...string) (int, error) {
	for _, key := range keys {
		raw, ok := optionalEnv(lookup, key)
		if !ok {
			continue
		}
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, &parseEnvError{key: key, message: "must be a positive integer"}
		}
		return parsed, nil
	}
	return 0, nil
}

func firstBoolEnv(lookup func(string) (string, bool), keys ...string) (bool, error) {
	for _, key := range keys {
		raw, ok := optionalEnv(lookup, key)
		if !ok {
			continue
		}
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return false, &parseEnvError{key: key, message: "must be true or false"}
		}
		return parsed, nil
	}
	return false, nil
}
