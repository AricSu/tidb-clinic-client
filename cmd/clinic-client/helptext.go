package main

import "strings"

func sharedMetricsHelp() string {
	return strings.TrimSpace(`
Run a Clinic metrics range query.

Required environment:
- CLINIC_API_KEY
- CLINIC_CLUSTER_ID

Context rules:
- CLINIC_ORG_TYPE defaults to cloud
- For TiDB Cloud targets, CLINIC_ORG_ID may be omitted and resolved from CLINIC_CLUSTER_ID
- For non-cloud targets, CLINIC_ORG_TYPE and CLINIC_ORG_ID are required
`)
}

func opWorkflowHelp(noun string) string {
	return strings.TrimSpace(`
` + noun + `

Required environment:
- CLINIC_API_KEY
- CLINIC_ORG_TYPE=op
- CLINIC_ORG_ID
- CLINIC_CLUSTER_ID

Notes:
- CLINIC_ITEM_ID is not required
- the CLI resolves an item ID automatically from catalog
`)
}

func cloudHelp(noun string) string {
	return strings.TrimSpace(`
` + noun + `

Required environment:
- CLINIC_API_KEY
- CLINIC_CLUSTER_ID

Context rules:
- CLINIC_ORG_TYPE defaults to cloud
- For TiDB Cloud targets, CLINIC_ORG_ID may be omitted and resolved from CLINIC_CLUSTER_ID
`)
}
