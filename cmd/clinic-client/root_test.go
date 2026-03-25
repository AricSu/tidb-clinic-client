package main

import (
	"bytes"
	"strings"
	"testing"
)

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
