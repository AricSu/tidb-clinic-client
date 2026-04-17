package cli

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type cloudEventDetailConfig struct {
	Base    cliConfig
	EventID string
}
type cloudClusterSearchConfig struct {
	Base        cliConfig
	Query       string
	ShowDeleted bool
	Limit       int
	Page        int
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
func loadClusterSearchConfig(lookup func(string) (string, bool), now func() time.Time) (cloudClusterSearchConfig, error) {
	baseURL, ok := optionalEnv(lookup, "CLINIC_API_BASE_URL")
	if !ok {
		baseURL = defaultBaseURL
	}
	apiKey, err := requiredEnv(lookup, "CLINIC_API_KEY")
	if err != nil {
		return cloudClusterSearchConfig{}, err
	}
	query, _ := optionalEnv(lookup, "CLINIC_CLUSTER_QUERY")
	clusterID, _ := optionalEnv(lookup, "CLINIC_CLUSTER_ID")
	if strings.TrimSpace(query) == "" && strings.TrimSpace(clusterID) == "" {
		return cloudClusterSearchConfig{}, errors.New("CLINIC_CLUSTER_QUERY or CLINIC_CLUSTER_ID is required")
	}
	showDeleted := true
	if raw, ok := optionalEnv(lookup, "CLINIC_SHOW_DELETED"); ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return cloudClusterSearchConfig{}, &parseEnvError{key: "CLINIC_SHOW_DELETED", message: "must be true or false"}
		}
		showDeleted = parsed
	}
	limit, err := optionalPositiveIntEnv(lookup, "CLINIC_LIMIT")
	if err != nil {
		return cloudClusterSearchConfig{}, err
	}
	page, err := optionalPositiveIntEnv(lookup, "CLINIC_PAGE")
	if err != nil {
		return cloudClusterSearchConfig{}, err
	}
	return cloudClusterSearchConfig{
		Base: cliConfig{
			BaseURL:   baseURL,
			APIKey:    apiKey,
			ClusterID: clusterID,
			Timeout:   defaultTimeout,
		},
		Query:       query,
		ShowDeleted: showDeleted,
		Limit:       limit,
		Page:        page,
	}, nil
}
