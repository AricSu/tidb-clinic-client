package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRootCommandIncludesPlatformGroupsOnly(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cloud") {
		t.Fatalf("expected root help to mention cloud, got=%q", output)
	}
	if !strings.Contains(output, "op") {
		t.Fatalf("expected root help to mention op, got=%q", output)
	}
	for _, unexpected := range []string{"metrics", "slowqueries", "logs", "configs"} {
		if strings.Contains(output, unexpected) {
			t.Fatalf("expected root help to hide %s, got=%q", unexpected, output)
		}
	}
}

func TestCloudMetricsCommandIncludesQueryRange(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "metrics", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "query-range") {
		t.Fatalf("expected cloud metrics help to mention query-range, got=%q", output)
	}
}

func TestCloudQueryRangeCommandRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudMetricsQueryRange: func() error {
			called = true
			return nil
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "metrics", "query-range"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud query-range handler to be called")
	}
}

func TestOldTopLevelMetricsCommandIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"metrics"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestOldTopLevelTransportCommandsAreUnknown(t *testing.T) {
	for _, name := range []string{"logs", "slowqueries", "configs"} {
		t.Run(name, func(t *testing.T) {
			cmd := newRootCommand()
			buf := &bytes.Buffer{}
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{name})

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected unknown command error")
			}
			if !strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("expected unknown command error, got=%v", err)
			}
		})
	}
}

func TestCloudCommandIncludesClusterDetail(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "cluster-detail") {
		t.Fatalf("expected cloud help to mention cluster-detail, got=%q", output)
	}
	if !strings.Contains(output, "events") {
		t.Fatalf("expected cloud help to mention events, got=%q", output)
	}
	if !strings.Contains(output, "metrics") {
		t.Fatalf("expected cloud help to mention metrics, got=%q", output)
	}
}

func TestCloudClusterDetailRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudClusterDetail: func() error {
			called = true
			return nil
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "cluster-detail"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cluster-detail handler to be called")
	}
}

func TestCloudEventsCommandIncludesQuery(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "events", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "query") {
		t.Fatalf("expected cloud events help to mention query, got=%q", output)
	}
}

func TestCloudEventsQueryRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudClusterDetail: func() error { return nil },
		runCloudEventsQuery: func() error {
			called = true
			return nil
		},
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "events", "query"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected events query handler to be called")
	}
}

func TestOPCommandIncludesExpectedGroups(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	output := buf.String()
	for _, expected := range []string{"metrics", "logs", "slowqueries", "configs"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected op help to mention %s, got=%q", expected, output)
		}
	}
	if !strings.Contains(output, "On-Premise") {
		t.Fatalf("expected op help to use On-Premise terminology, got=%q", output)
	}
}

func TestOPMetricsCommandIncludesQueryRange(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "metrics", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(buf.String(), "query-range") {
		t.Fatalf("expected op metrics help to mention query-range, got=%q", buf.String())
	}
}

func TestOPMetricsQueryRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runOPMetricsQueryRange: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "metrics", "query-range"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected op metrics query handler to be called")
	}
}

func TestOPLogsSearchRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runOPLogsSearch: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "logs"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected op logs search handler to be called")
	}
}

func TestOPLogsHelpIncludesRequiredContext(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "logs", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	output := buf.String()
	for _, expected := range []string{"CLINIC_API_KEY", "CLINIC_ORG_ID", "CLINIC_CLUSTER_ID"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected op logs help to mention %s, got=%q", expected, output)
		}
	}
}

func TestOPLogsNestedSearchIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "logs", "search"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestOPSlowQueriesQueryRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runOPSlowQueriesQuery: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "slowqueries"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected op slowqueries query handler to be called")
	}
}

func TestOPSlowQueriesNestedQueryIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "slowqueries", "query"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestOPConfigsGetRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runOPConfigsGet: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "configs"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected op configs get handler to be called")
	}
}

func TestOPConfigsNestedGetIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "configs", "get"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestRootHelpDoesNotExposeCatalogCommand(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if strings.Contains(buf.String(), "catalog") {
		t.Fatalf("expected root help to hide catalog, got=%q", buf.String())
	}
}

func TestCatalogCommandIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"catalog", "list"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestOldTopLevelLogsCommandIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"logs"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestOldTopLevelSlowQueriesCommandIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"slowqueries"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestOldTopLevelConfigsCommandIsUnknown(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"configs"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected unknown command error, got=%v", err)
	}
}

func TestCloudEventsCommandIncludesDetail(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "events", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "detail") {
		t.Fatalf("expected cloud events help to mention detail, got=%q", output)
	}
}

func TestCloudEventsDetailRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudEventsDetail: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "events", "detail"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected events detail handler to be called")
	}
}

func TestCloudTopSQLCommandIncludesSummary(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "topsql", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(buf.String(), "summary") {
		t.Fatalf("expected cloud topsql help to mention summary, got=%q", buf.String())
	}
}

func TestCloudTopSQLSummaryRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudTopSQLSummary: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "topsql", "summary"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected topsql summary handler to be called")
	}
}

func TestCloudMetricsHelpIncludesRequiredContext(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "metrics", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	output := buf.String()
	for _, expected := range []string{"CLINIC_API_KEY", "CLINIC_CLUSTER_ID"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected cloud metrics help to mention %s, got=%q", expected, output)
		}
	}
}

func TestOPMetricsHelpIncludesRequiredContext(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"op", "metrics", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	output := buf.String()
	for _, expected := range []string{"CLINIC_API_KEY", "CLINIC_ORG_ID", "CLINIC_CLUSTER_ID"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected op metrics help to mention %s, got=%q", expected, output)
		}
	}
}

func TestCloudClusterDetailHelpIncludesRequiredContext(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "cluster-detail", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	output := buf.String()
	for _, expected := range []string{"CLINIC_API_KEY", "CLINIC_CLUSTER_ID"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected cloud cluster-detail help to mention %s, got=%q", expected, output)
		}
	}
}

func TestCloudSlowQueriesCommandIncludesAllLeaves(t *testing.T) {
	cmd := newRootCommand()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "slowqueries", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	output := buf.String()
	for _, expected := range []string{"top", "list", "detail"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected cloud slowqueries help to mention %s, got=%q", expected, output)
		}
	}
}

func TestCloudSlowQueriesTopRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudTopSlowQueries: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "slowqueries", "top"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected slowqueries top handler to be called")
	}
}

func TestCloudSlowQueriesListRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudSlowQueriesList: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "slowqueries", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected slowqueries list handler to be called")
	}
}

func TestCloudSlowQueriesDetailRunsInjectedHandler(t *testing.T) {
	var called bool
	cmd := newRootCommandWithDeps(commandDeps{
		runCloudSlowQueriesDetail: func() error { called = true; return nil },
	})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud", "slowqueries", "detail"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected slowqueries detail handler to be called")
	}
}
