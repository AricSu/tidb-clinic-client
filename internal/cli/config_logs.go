package cli

import (
	"strconv"
	"strings"
	"time"
)

type cloudLogsMode string

const (
	cloudLogsModeLabels      cloudLogsMode = "labels"
	cloudLogsModeLabelValues cloudLogsMode = "label-values"
	cloudLogsModeQueryRange  cloudLogsMode = "query-range"
)

type cloudLogsConfig struct {
	Base      cliConfig
	Mode      cloudLogsMode
	Query     string
	Limit     int
	Direction string
	LabelName string
}

type cloudLogsFlagInputs struct {
	Query        string
	QuerySet     bool
	Limit        int
	LimitSet     bool
	Direction    string
	DirectionSet bool
	LabelName    string
	LabelNameSet bool
}

var activeCloudLogsFlagInputs cloudLogsFlagInputs

func pushCloudLogsFlagInputs(next cloudLogsFlagInputs) func() {
	previous := activeCloudLogsFlagInputs
	activeCloudLogsFlagInputs = next
	return func() {
		activeCloudLogsFlagInputs = previous
	}
}

func loadCloudLogsConfig(lookup func(string) (string, bool), now func() time.Time) (cloudLogsConfig, error) {
	base, err := loadConfigFromEnv(lookup, now)
	if err != nil {
		return cloudLogsConfig{}, err
	}

	labelName, hasLabelSignal, err := resolveCloudLogsLabelInput(lookup)
	if err != nil {
		return cloudLogsConfig{}, err
	}
	query, limit, direction, hasQuerySignal, err := resolveCloudLogsQueryInputs(lookup)
	if err != nil {
		return cloudLogsConfig{}, err
	}
	if hasLabelSignal && hasQuerySignal {
		return cloudLogsConfig{}, &parseEnvError{key: "cloud-logs", message: "cannot combine label-values inputs with query-range inputs"}
	}
	if hasLabelSignal {
		return cloudLogsConfig{
			Base:      base,
			Mode:      cloudLogsModeLabelValues,
			LabelName: labelName,
		}, nil
	}
	if hasQuerySignal {
		return cloudLogsConfig{
			Base:      base,
			Mode:      cloudLogsModeQueryRange,
			Query:     query,
			Limit:     limit,
			Direction: direction,
		}, nil
	}
	return cloudLogsConfig{Base: base, Mode: cloudLogsModeLabels}, nil
}

func resolveCloudLogsLabelInput(lookup func(string) (string, bool)) (string, bool, error) {
	if activeCloudLogsFlagInputs.LabelNameSet {
		labelName := strings.TrimSpace(activeCloudLogsFlagInputs.LabelName)
		if labelName == "" {
			return "", false, &parseEnvError{key: "--label", message: "is required"}
		}
		return labelName, true, nil
	}
	labelName, ok := optionalEnv(lookup, "CLINIC_LOKI_LABEL")
	return labelName, ok, nil
}

func resolveCloudLogsQueryInputs(lookup func(string) (string, bool)) (string, int, string, bool, error) {
	if activeCloudLogsFlagInputs.QuerySet || activeCloudLogsFlagInputs.LimitSet || activeCloudLogsFlagInputs.DirectionSet {
		query := strings.TrimSpace(activeCloudLogsFlagInputs.Query)
		if query == "" {
			return "", 0, "", false, &parseEnvError{key: "--query", message: "is required when using log query inputs"}
		}
		return query, activeCloudLogsFlagInputs.Limit, strings.TrimSpace(activeCloudLogsFlagInputs.Direction), true, nil
	}

	query, querySet := optionalEnv(lookup, "CLINIC_LOKI_QUERY")
	limitRaw, limitSet := optionalEnv(lookup, "CLINIC_LOKI_LIMIT")
	direction, directionSet := optionalEnv(lookup, "CLINIC_LOKI_DIRECTION")
	if !querySet && !limitSet && !directionSet {
		return "", 0, "", false, nil
	}
	if !querySet {
		return "", 0, "", false, &parseEnvError{key: "CLINIC_LOKI_QUERY", message: "is required when using log query inputs"}
	}
	limit := 0
	if limitSet {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed <= 0 {
			return "", 0, "", false, &parseEnvError{key: "CLINIC_LOKI_LIMIT", message: "must be a positive integer"}
		}
		limit = parsed
	}
	return query, limit, direction, true, nil
}
