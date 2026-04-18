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
	use, short, long, runID string
	commands                []commandSpec
	groups                  []groupSpec
}

func defaultApp() App {
	lookupEnv := lookupEnvWithDotEnv(os.LookupEnv, ".env")
	bind := func(run commandRunner) func() error {
		return func() error { return run(lookupEnv, time.Now, log.New(os.Stderr, "", 0), os.Stdout) }
	}
	return App{
		"cluster":                       bind(runClusterInfo),
		"metrics.query":                 bind(runMetricsQueryRange),
		"metrics.compile":               bind(runMetricsCompile),
		"slowquery":                     bind(runSlowQuery),
		"op-pkgs.list":                  bind(runCollectedDataList),
		"op-pkgs.download":              bind(runCollectedDataDownload),
		"cloud-events.search":           bind(runCloudEventsQuery),
		"cloud-logs.search":             bind(runCloudLogs),
		"cloud-profilings.list":         bind(runCloudProfilingGroups),
		"cloud-profilings.download":     bind(runCloudProfilingDownload),
		"cloud-plan-replayers.list":     bind(runCloudDiagnosticPlan),
		"cloud-plan-replayers.download": bind(runCloudDiagnosticDownload),
		"cloud-oom-records.list":        bind(runCloudDiagnosticOOM),
		"cloud-oom-records.download":    bind(runCloudDiagnosticDownload),
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
		Long:          helpBlock("Task-first TiDB Clinic CLI.", "Preferred command paths:", "- cluster", "- metrics query", "- metrics compile", "- slowquery", "- op-pkgs list", "- op-pkgs download", "- cloud-events search", "- cloud-logs search", "- cloud-profilings list", "- cloud-profilings download", "- cloud-plan-replayers list", "- cloud-plan-replayers download", "- cloud-oom-records list", "- cloud-oom-records download"),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	for _, group := range capabilityGroups() {
		root.AddCommand(newGroupCommand(group, deps))
	}
	return root
}

func newGroupCommand(group groupSpec, deps App) *cobra.Command {
	run := deps[group.runID]
	var groupCmd *cobra.Command
	groupCmd = &cobra.Command{
		Use:   group.use,
		Short: group.short,
		Long:  group.long,
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			if run == nil {
				return groupCmd.Help()
			}
			return run()
		},
	}
	for _, spec := range group.commands {
		groupCmd.AddCommand(newLeafCommand(spec, deps))
	}
	for _, child := range group.groups {
		groupCmd.AddCommand(newGroupCommand(child, deps))
	}
	configureCommand(groupCmd, group.runID)
	return groupCmd
}

func newLeafCommand(spec commandSpec, deps App) *cobra.Command {
	run := deps[spec.runID]
	leafCmd := &cobra.Command{
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
	configureCommand(leafCmd, spec.runID)
	return leafCmd
}

func configureCommand(cmd *cobra.Command, runID string) {
	if cmd == nil {
		return
	}
	switch runID {
	case "metrics.compile":
		configureMetricsCompileCommand(cmd)
	case "slowquery":
		configureSlowQueryCommand(cmd)
	case "cloud-events.search":
		configureEventsCommand(cmd)
	case "cloud-logs.search":
		configureCloudLogsCommand(cmd)
	}
}

func configureMetricsCompileCommand(cmd *cobra.Command) {
	var flags metricsCompileFlagInputs
	cmd.Flags().StringVar(&flags.ExprDescription, "expr-description", "", "Human-readable explanation for the expr")
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		flags.ExprDescriptionSet = cmd.Flags().Changed("expr-description")
		restore := pushMetricsCompileFlagInputs(flags)
		defer restore()
		if originalRunE == nil {
			return nil
		}
		return originalRunE(cmd, args)
	}
}

func configureEventsCommand(cmd *cobra.Command) {
	var flags eventFlagInputs
	cmd.Flags().StringVar(&flags.Name, "name", "", "Activity name search term")
	cmd.Flags().StringVar(&flags.Severity, "severity", "", "Event severity: info, warning, debug, critical")
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		flags.NameSet = cmd.Flags().Changed("name")
		flags.SeveritySet = cmd.Flags().Changed("severity")
		restore := pushEventFlagInputs(flags)
		defer restore()
		if originalRunE == nil {
			return nil
		}
		return originalRunE(cmd, args)
	}
}

func configureSlowQueryCommand(cmd *cobra.Command) {
	var flags slowQueryFlagInputs
	cmd.Flags().StringVar(&flags.OrderBy, "order-by", "", "Slow query sort field")
	cmd.Flags().IntVar(&flags.Limit, "limit", 0, "Max slow query records to return")
	cmd.Flags().BoolVar(&flags.Desc, "desc", false, "Sort slow queries in descending order")
	cmd.Flags().StringVar(&flags.Digest, "digest", "", "Slow query digest to sample")
	cmd.Flags().StringVar(&flags.Fields, "fields", "", "Comma-separated sample fields to request")
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		flags.OrderBySet = cmd.Flags().Changed("order-by")
		flags.LimitSet = cmd.Flags().Changed("limit")
		flags.DescSet = cmd.Flags().Changed("desc")
		flags.DigestSet = cmd.Flags().Changed("digest")
		flags.FieldsSet = cmd.Flags().Changed("fields")
		restore := pushSlowQueryFlagInputs(flags)
		defer restore()
		if originalRunE == nil {
			return nil
		}
		return originalRunE(cmd, args)
	}
}

func configureCloudLogsCommand(cmd *cobra.Command) {
	var flags cloudLogsFlagInputs
	cmd.Flags().StringVar(&flags.Query, "query", "", "LogQL query")
	cmd.Flags().IntVar(&flags.Limit, "limit", 0, "Max log lines to return")
	cmd.Flags().StringVar(&flags.Direction, "direction", "", "Log direction: forward or backward")
	cmd.Flags().StringVar(&flags.LabelName, "label", "", "Log label name")
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		flags.QuerySet = cmd.Flags().Changed("query")
		flags.LimitSet = cmd.Flags().Changed("limit")
		flags.DirectionSet = cmd.Flags().Changed("direction")
		flags.LabelNameSet = cmd.Flags().Changed("label")
		restore := pushCloudLogsFlagInputs(flags)
		defer restore()
		if originalRunE == nil {
			return nil
		}
		return originalRunE(cmd, args)
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
			use: "cluster", short: "Get cluster detail as raw JSON", long: clusterHelp(), runID: "cluster",
		},
		{
			use: "metrics", short: "Query metrics", long: metricsGroupHelp(),
			commands: []commandSpec{
				cmdLong("query", "Query metrics over a time range", metricsQueryHelp(), "metrics.query"),
				cmdLong("compile", "Query metrics and analyze them with compiler-rs", metricsCompileHelp(), "metrics.compile"),
			},
		},
		{
			use: "slowquery", short: "Query slow queries for the current cluster and time range", long: slowQueryHelp(), runID: "slowquery",
		},
		{
			use: "op-pkgs", short: "Work with on-premise collected-data packages", long: opPkgHelp(),
			commands: []commandSpec{
				cmdLong("list", "List on-premise collected-data packages", opPkgsHelp(), "op-pkgs.list"),
				cmdLong("download", "Download one on-premise collected-data package", opPkgDownloadHelp(), "op-pkgs.download"),
			},
		},
		{
			use: "cloud-events", short: "Search cloud cluster events", long: cloudEventHelp(),
			commands: []commandSpec{
				cmdLong("search", "Search cloud cluster events", cloudEventsHelp(), "cloud-events.search"),
			},
		},
		{
			use: "cloud-logs", short: "Search cloud logs", long: cloudLogsGroupHelp(),
			commands: []commandSpec{
				cmdLong("search", "Search cloud logs with automatic mode selection", cloudLogsHelp(), "cloud-logs.search"),
			},
		},
		{
			use: "cloud-profilings", short: "Work with cloud profiling snapshots", long: cloudProfilingsGroupHelp(),
			commands: []commandSpec{
				cmdLong("list", "List cloud profiling snapshots", cloudProfilingsListHelp(), "cloud-profilings.list"),
				cmdLong("download", "Download one cloud profiling artifact", cloudProfilingDownloadHelp(), "cloud-profilings.download"),
			},
		},
		{
			use: "cloud-plan-replayers", short: "Work with cloud plan replayer artifacts", long: cloudPlanReplayersGroupHelp(),
			commands: []commandSpec{
				cmdLong("list", "List cloud plan replayer artifacts", cloudPlanReplayersHelp(), "cloud-plan-replayers.list"),
				cmdLong("download", "Download one cloud diagnostic artifact", cloudDiagnosticDownloadHelp(), "cloud-plan-replayers.download"),
			},
		},
		{
			use: "cloud-oom-records", short: "Work with cloud OOM record artifacts", long: cloudOOMRecordsGroupHelp(),
			commands: []commandSpec{
				cmdLong("list", "List cloud OOM record artifacts", cloudOOMRecordsHelp(), "cloud-oom-records.list"),
				cmdLong("download", "Download one cloud diagnostic artifact", cloudDiagnosticDownloadHelp(), "cloud-oom-records.download"),
			},
		},
	}
}

func helpBlock(lines ...string) string { return strings.TrimSpace(strings.Join(lines, "\n")) }

func clusterHelp() string {
	return cloudHelp("Get cluster detail as raw JSON.")
}

func metricsGroupHelp() string {
	return cloudHelp("Query or compile metrics.")
}

func metricsQueryHelp() string {
	return helpBlock(
		"Run a Clinic metrics range query.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Metrics query inputs:",
		"- CLINIC_METRICS_QUERY",
		"- CLINIC_RANGE_STEP",
	)
}

func metricsCompileHelp() string {
	return helpBlock(
		"Query Clinic metrics and immediately analyze the returned series with compiler-rs.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Metrics query inputs:",
		"- CLINIC_METRICS_QUERY",
		"- CLINIC_RANGE_STEP",
		"",
		"Optional environment:",
		"- CLINIC_METRICS_EXPR_DESCRIPTION",
		"",
		"Flags:",
		"- --expr-description for a human-readable explanation of the expr",
		"",
		"Behavior:",
		"- runs the same metrics range query as `metrics query`",
		"- automatically chooses compiler-rs line or group input based on the returned series layout",
		"- prints the compiler-rs analysis JSON",
	)
}

func opPkgHelp() string {
	return cloudHelp("Work with on-premise / OP collected-data packages.")
}

func slowQueryHelp() string {
	return helpBlock(
		"Query slow queries from the retained Clinic data plane.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Optional environment:",
		"- CLINIC_SLOWQUERY_ORDER_BY",
		"- CLINIC_SLOWQUERY_LIMIT",
		"- CLINIC_SLOWQUERY_DESC",
		"- CLINIC_SLOWQUERY_DIGEST",
		"- CLINIC_SLOWQUERY_FIELDS",
		"",
		"Flags:",
		"- --order-by for server-side sorting",
		"- --limit for max returned records",
		"- --desc for descending sort order",
		"- --digest to switch from record search to sample lookup",
		"- --fields for comma-separated sample fields",
		"",
		"Notes:",
		"- default order-by is `query_time`",
		"- default limit is `100`",
		"- shared capability for both cloud and non-cloud / OP clusters",
		"- cloud queries go directly to the retained slowquery API and do not need item selection",
		"- non-cloud / OP queries automatically select a matching collected bundle in the current time window",
		"- when selecting an OP bundle, slowquery prefers bundles containing the log.slow collector when available",
		"- without `--digest`, `slowquery` returns retained slow query records",
		"- with `--digest`, `slowquery` returns concrete sample rows for that digest",
		"- OP sample queries keep source_ref as item-scoped provenance; connection_id is a sample locator",
	)
}

func opPkgDownloadHelp() string {
	return helpBlock(
		"Download one collected data bundle for non-cloud / OP deployments.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Optional environment:",
		"- CLINIC_OUTPUT_PATH for data download",
		"",
		"Notes:",
		"- cloud clusters are not supported",
		"- bundle selection is automatic inside the SDK",
		"- when no range is provided, download chooses the latest collected bundle",
	)
}

func cloudDiagnosticDownloadHelp() string {
	return helpBlock(
		"Download one cloud diagnostic artifact by storage key.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"- CLINIC_DIAGNOSTIC_KEY",
		"",
		"Optional environment:",
		"- CLINIC_OUTPUT_PATH for artifact download",
	)
}

func cloudProfilingDownloadHelp() string {
	return helpBlock(
		"Download one cloud profiling artifact.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"- CLINIC_PROFILE_TS",
		"- CLINIC_PROFILE_TYPE",
		"- CLINIC_PROFILE_COMPONENT",
		"- CLINIC_PROFILE_ADDRESS",
		"",
		"Optional environment:",
		"- CLINIC_PROFILE_DATA_FORMAT",
		"- CLINIC_OUTPUT_PATH for artifact download",
	)
}

func cloudHelp(noun string) string {
	return helpBlock(
		noun,
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
	)
}

func cloudLogsGroupHelp() string {
	return cloudHelp("Search cloud logs.")
}

func cloudLogsHelp() string {
	return helpBlock(
		"Search cloud logs with automatic mode selection.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Mode selection:",
		"- if --label or CLINIC_LOKI_LABEL is present, query label values",
		"- else if any query-range inputs are present, run a ranged log query",
		"- else, list available log labels",
		"",
		"Flags:",
		"- --label for label-values",
		"- --query for the LogQL query",
		"- --limit for max returned log lines",
		"- --direction with one of: forward, backward",
		"",
		"Notes:",
		"- cloud only",
		"- --label cannot be combined with query-range inputs",
	)
}

func cloudProfilingsGroupHelp() string {
	return cloudHelp("Work with cloud profiling snapshots.")
}

func cloudProfilingsListHelp() string {
	return helpBlock(
		"List cloud profiling groups in the current time window.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Notes:",
		"- cloud only",
		"- profiling artifact download uses cloud-profilings download",
	)
}

func cloudEventHelp() string {
	return cloudHelp("Search cloud cluster events.")
}

func cloudEventsHelp() string {
	return helpBlock(
		"Search cluster events for cloud clusters.",
		"",
		"Output:",
		"- raw JSON from the activityhub events API",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Flags:",
		"- --name for activity name search",
		"- --severity with one of: info, warning, debug, critical",
		"",
		"Notes:",
		"- cloud only",
		"- non-cloud deployments are not supported",
	)
}

func cloudPlanReplayersGroupHelp() string {
	return cloudHelp("Work with cloud plan replayer artifacts.")
}

func cloudPlanReplayersHelp() string {
	return cloudHelp("List cloud plan replayer artifacts.")
}

func cloudOOMRecordsGroupHelp() string {
	return cloudHelp("Work with cloud OOM record artifacts.")
}

func cloudOOMRecordsHelp() string {
	return cloudHelp("List cloud OOM record artifacts.")
}

func opPkgsHelp() string {
	return helpBlock(
		"List collected data packages for non-cloud / OP deployments.",
		"",
		"Required environment:",
		"- CLINIC_PORTAL_URL",
		"- CLINIC_API_KEY for clinic.pingcap.com",
		"- CLINIC_CN_API_KEY for clinic.pingcap.com.cn",
		"",
		"Output:",
		"- one line per collected bundle with item id, file name, time range, and available data types",
		"",
		"Notes:",
		"- cloud clusters are not supported",
	)
}
