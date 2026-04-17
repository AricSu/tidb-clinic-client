package cli

import (
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type App map[string]func() error
type commandRunner func(func(string) (string, bool), func() time.Time, *log.Logger, io.Writer) error
type commandSpec struct {
	use, short, long, runID string
}
type groupSpec struct {
	use, short, long string
	commands         []commandSpec
}

func defaultApp() App {
	lookupEnv := lookupEnvWithDotEnv(os.LookupEnv, ".env")
	bind := func(run commandRunner) func() error {
		return func() error { return run(lookupEnv, time.Now, log.Default(), os.Stdout) }
	}
	return App{
		"metrics.query-range":   bind(runMetricsQueryRange),
		"metrics.query-instant": bind(runMetricsQueryInstant),
		"metrics.query-series":  bind(runMetricsQuerySeries),
		"sql.slowquery-records": bind(runRetainedSlowQueriesQuery),
		"logs.search":           bind(runRetainedLogsSearch),
		"configs.get":           bind(runRetainedConfigsGet),
		"clusters.detail":       bind(runClusterDetail),
		"clusters.search":       bind(runClusterSearch),
		"clusters.topology":     bind(runClusterTopology),
		"sql.query":             bind(runCloudDataProxyQuery),
		"sql.schema":            bind(runCloudDataProxySchema),
		"sql.statements":        bind(runCapabilitySQLStatements),
		"clusters.events":       bind(runCloudEventsQuery),
		"clusters.event-detail": bind(runCloudEventsDetail),
		"logs.query":            bind(runCloudLokiQuery),
		"logs.query-range":      bind(runCloudLokiQueryRange),
		"logs.labels":           bind(runCloudLokiLabels),
		"logs.label-values":     bind(runCloudLokiLabelValues),
		"profiling.groups":      bind(runCloudProfilingGroups),
		"profiling.detail":      bind(runCloudProfilingDetail),
		"profiling.download":    bind(runCloudProfilingDownload),
		"diagnostics.plan":      bind(runCloudDiagnosticPlan),
		"diagnostics.oom":       bind(runCloudDiagnosticOOM),
		"diagnostics.download":  bind(runCloudDiagnosticDownload),
		"sql.topsql":            bind(runCloudTopSQLSummary),
		"sql.slowquery-top":     bind(runCloudTopSlowQueries),
		"sql.slowquery-samples": bind(runCloudSlowQueriesList),
		"sql.slowquery-detail":  bind(runCloudSlowQueriesDetail),
		"capabilities.discover": bind(runCapabilityDiscover),
	}
}
func lookupEnvWithDotEnv(base func(string) (string, bool), path string) func(string) (string, bool) {
	dotenv := parseDotEnvFile(path)
	return func(key string) (string, bool) {
		if value, ok := base(key); ok && strings.TrimSpace(value) != "" {
			return value, true
		}
		value, ok := dotenv[key]
		return value, ok
	}
}
func parseDotEnvFile(path string) map[string]string {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := make(map[string]string)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		sep := strings.IndexByte(line, '=')
		if sep <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:sep])
		if key == "" {
			continue
		}
		if value := parseDotEnvValue(line[sep+1:]); strings.TrimSpace(value) != "" {
			out[key] = value
		}
	}
	return out
}
func parseDotEnvValue(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return strings.TrimSpace(unquoted)
		}
		return strings.TrimSpace(value[1 : len(value)-1])
	}
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}
func NewCommand() *cobra.Command { return newRootCommandWithApp(defaultApp()) }
func newRootCommandWithApp(deps App) *cobra.Command {
	cobra.EnableCommandSorting = false
	root := &cobra.Command{
		Use:           "clinic-client",
		Short:         "TiDB Clinic CLI",
		Long:          helpBlock("Capability-first TiDB Clinic CLI.", "Preferred command paths:", "- clusters", "- metrics", "- logs", "- sql", "- configs", "- profiling", "- diagnostics", "- capabilities"),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	for _, group := range capabilityGroups() {
		root.AddCommand(newGroupCommand(group, deps))
	}
	return root
}
func newGroupCommand(group groupSpec, deps App) *cobra.Command {
	cmd := &cobra.Command{Use: group.use, Short: group.short, Long: group.long}
	for _, spec := range group.commands {
		cmd.AddCommand(newLeafCommand(spec, deps))
	}
	return cmd
}
func newLeafCommand(spec commandSpec, deps App) *cobra.Command {
	run := deps[spec.runID]
	return &cobra.Command{
		Use:   spec.use,
		Short: spec.short,
		Long:  spec.long,
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if run == nil {
				return nil
			}
			return run()
		},
	}
}
func cmd(use, short, runID string) commandSpec {
	return commandSpec{use: use, short: short, runID: runID}
}
func cmdLong(use, short, long, runID string) commandSpec {
	return commandSpec{use: use, short: short, long: long, runID: runID}
}
func capabilityGroups() []groupSpec {
	return []groupSpec{
		{
			use: "clusters", short: "Run capability-first cluster commands", long: cloudHelp("Run capability-first cluster commands."),
			commands: []commandSpec{
				cmd("search", "Search clusters", "clusters.search"),
				cmd("detail", "Get cluster detail", "clusters.detail"),
				cmd("topology", "Get cluster topology", "clusters.topology"),
				cmd("events", "Query cluster events", "clusters.events"),
				cmd("event-detail", "Get one cluster event detail", "clusters.event-detail"),
			},
		},
		{
			use: "metrics", short: "Run capability-first metrics queries", long: sharedMetricsHelp(),
			commands: []commandSpec{
				cmd("query-range", "Query metrics over a time range", "metrics.query-range"),
				cmd("query-instant", "Query metrics at one point in time", "metrics.query-instant"),
				cmd("query-series", "Discover metric label sets over a time range", "metrics.query-series"),
			},
		},
		{
			use: "logs", short: "Run capability-first log queries", long: cloudHelp("Run capability-first log queries."),
			commands: []commandSpec{
				cmd("query", "Run an instant log query", "logs.query"),
				cmd("query-range", "Run a ranged log query", "logs.query-range"),
				cmd("labels", "List log labels", "logs.labels"),
				cmd("label-values", "List values for one log label", "logs.label-values"),
				cmd("search", "Search collected log records with automatic item selection", "logs.search"),
			},
		},
		{
			use: "sql", short: "Run capability-first SQL analytics commands", long: cloudHelp("Run capability-first SQL analytics commands."),
			commands: []commandSpec{
				cmd("query", "Run ad-hoc SQL via Data Proxy", "sql.query"),
				cmd("schema", "Fetch SQL analytics schema", "sql.schema"),
				cmd("topsql", "Get TopSQL summary", "sql.topsql"),
				cmd("slowquery-top", "Get aggregated slow-query summary", "sql.slowquery-top"),
				cmdLong("slowquery-samples", "List slow-query samples", collectedWorkflowHelp("List slow-query samples. The output includes id and item_id so a sample can be used directly with slowquery-detail."), "sql.slowquery-samples"),
				cmdLong("slowquery-detail", "Get one slow-query detail", slowQueryDetailHelp(), "sql.slowquery-detail"),
				cmd("statements", "Run SQL statements queries", "sql.statements"),
				cmd("slowquery-records", "Query collected slow-query records with automatic item selection", "sql.slowquery-records"),
			},
		},
		{
			use: "configs", short: "Run capability-first config retrieval", long: collectedWorkflowHelp("Fetch tiup-cluster collected config snapshots through the configs client."),
			commands: []commandSpec{
				cmd("get", "Fetch config snapshots with automatic item selection", "configs.get"),
			},
		},
		{
			use: "profiling", short: "Run TiDB Cloud continuous profiling commands", long: cloudHelp("Run TiDB Cloud continuous profiling commands."),
			commands: []commandSpec{
				cmd("groups", "List profiling groups in a time window", "profiling.groups"),
				cmd("detail", "Get profiling detail for one snapshot timestamp", "profiling.detail"),
				cmd("download", "Fetch a profiling artifact to disk", "profiling.download"),
			},
		},
		{
			use: "diagnostics", short: "Run TiDB Cloud diagnostic artifact commands", long: cloudHelp("Run TiDB Cloud diagnostic artifact commands."),
			commands: []commandSpec{
				cmd("plan-replayer", "List plan replayer artifacts", "diagnostics.plan"),
				cmd("oom-record", "List OOM record artifacts", "diagnostics.oom"),
				cmd("download", "Download one diagnostic artifact by storage key", "diagnostics.download"),
			},
		},
		{
			use: "capabilities", short: "Inspect capability availability for one cluster", long: cloudHelp("Inspect capability availability for one cluster."),
			commands: []commandSpec{
				cmd("discover", "Resolve the capability contract for one cluster", "capabilities.discover"),
			},
		},
	}
}
func helpBlock(lines ...string) string { return strings.TrimSpace(strings.Join(lines, "\n")) }
func sharedMetricsHelp() string {
	return helpBlock(
		"Run a Clinic metrics query.",
		"",
		"Required environment:",
		"- CLINIC_API_KEY",
		"- CLINIC_CLUSTER_ID",
		"",
		"Metrics query inputs:",
		"- CLINIC_METRICS_QUERY for query-range and query-instant",
		"- CLINIC_METRICS_MATCH for query-series",
		"- CLINIC_QUERY_TIME for query-instant",
		"- CLINIC_RANGE_START / CLINIC_RANGE_END / CLINIC_RANGE_STEP",
	)
}
func collectedWorkflowHelp(noun string) string {
	return helpBlock(
		noun,
		"",
		"Required environment:",
		"- CLINIC_API_KEY",
		"- CLINIC_CLUSTER_ID",
		"",
		"Notes:",
		"- item selection is automatic inside the SDK",
		"- set CLINIC_VERBOSE_LOGS=true for request lifecycle logs",
	)
}
func cloudHelp(noun string) string {
	return helpBlock(noun, "", "Required environment:", "- CLINIC_API_KEY", "- CLINIC_CLUSTER_ID")
}
func slowQueryDetailHelp() string {
	return helpBlock(
		"Get one slow-query detail.",
		"",
		"Required environment:",
		"- CLINIC_API_KEY",
		"- CLINIC_CLUSTER_ID",
		"",
		"Lookup modes:",
		"- preferred: CLINIC_SLOWQUERY_ID",
		"- tiup uses CLINIC_RANGE_START and CLINIC_RANGE_END",
		"- cloud uses DIGEST + CONNECTION_ID + TIMESTAMP",
	)
}
