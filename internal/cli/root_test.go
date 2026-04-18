package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpShowsFinalTopLevelCommands(t *testing.T) {
	cmd := newRootCommandWithApp(App{})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	help := buf.String()
	for _, name := range []string{"cluster", "metrics", "slowquery", "op-pkgs", "cloud-events", "cloud-logs", "cloud-profilings", "cloud-plan-replayers", "cloud-oom-records"} {
		if !strings.Contains(help, "  "+name) {
			t.Fatalf("expected help to include %q, got:\n%s", name, help)
		}
	}
}

func TestClusterCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cluster": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cluster"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cluster handler to be called")
	}
}

func TestMetricsQueryCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"metrics.query": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"metrics", "query"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected metrics query handler to be called")
	}
}

func TestMetricsCompileCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"metrics.compile": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"metrics", "compile"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected metrics compile handler to be called")
	}
}

func TestSlowQueryCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"slowquery": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"slowquery"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected slowquery handler to be called")
	}
}

func TestSlowQueryCommandAcceptsFlags(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"slowquery": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"slowquery", "--order-by", "queryTime", "--limit", "10", "--desc", "--digest", "digest-1", "--fields", "query,timestamp"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected slowquery handler to be called")
	}
}

func TestSlowQueryHelpShowsSharedCapabilityNotes(t *testing.T) {
	cmd := newRootCommandWithApp(App{})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"slowquery", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	help := buf.String()
	for _, expected := range []string{
		"retained Clinic data plane",
		"CLINIC_SLOWQUERY_ORDER_BY",
		"--order-by",
		"shared capability for both cloud and non-cloud / OP clusters",
		"cloud queries go directly to the retained slowquery API and do not need item selection",
		"when selecting an OP bundle, slowquery prefers bundles containing the log.slow collector when available",
		"without `--digest`, `slowquery` returns retained slow query records",
		"with `--digest`, `slowquery` returns concrete sample rows for that digest",
		"CLINIC_SLOWQUERY_DIGEST",
		"--digest",
		"CLINIC_SLOWQUERY_FIELDS",
		"OP sample queries keep source_ref as item-scoped provenance",
	} {
		if !strings.Contains(help, expected) {
			t.Fatalf("expected help to include %q, got:\n%s", expected, help)
		}
	}
}

func TestCloudEventSearchCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-events.search": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-events", "search"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-events search handler to be called")
	}
}

func TestCloudEventSearchCommandAcceptsFlags(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-events.search": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-events", "search", "--name", "%backup%", "--severity", "critical"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-events search handler to be called")
	}
}

func TestCloudEventSearchHelpShowsFilterInputsAndPlatformNotes(t *testing.T) {
	cmd := newRootCommandWithApp(App{})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"cloud-events", "search", "--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	help := buf.String()
	for _, expected := range []string{
		"raw JSON from the activityhub events API",
		"--name",
		"--severity",
		"info, warning, debug, critical",
		"cloud only",
		"non-cloud deployments are not supported",
	} {
		if !strings.Contains(help, expected) {
			t.Fatalf("expected help to include %q, got:\n%s", expected, help)
		}
	}
}

func TestCloudLogsSearchCommandAcceptsQueryFlags(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-logs.search": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-logs", "search", "--query", `{container="tidb"}`, "--limit", "10", "--direction", "backward"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-logs search handler to be called")
	}
}

func TestCloudLogsSearchCommandAcceptsLabelFlag(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-logs.search": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-logs", "search", "--label", "container"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-logs search handler to be called")
	}
}

func TestCloudProfilingsListCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-profilings.list": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-profilings", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-profilings list handler to be called")
	}
}

func TestCloudProfilingsDownloadCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-profilings.download": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-profilings", "download"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-profilings download handler to be called")
	}
}

func TestCloudPlanReplayersListCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-plan-replayers.list": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-plan-replayers", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-plan-replayers list handler to be called")
	}
}

func TestCloudPlanReplayersDownloadCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-plan-replayers.download": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-plan-replayers", "download"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-plan-replayers download handler to be called")
	}
}

func TestCloudOOMRecordsListCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-oom-records.list": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-oom-records", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-oom-records list handler to be called")
	}
}

func TestCloudOOMRecordsDownloadCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"cloud-oom-records.download": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"cloud-oom-records", "download"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected cloud-oom-records download handler to be called")
	}
}

func TestOpPkgListCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"op-pkgs.list": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"op-pkgs", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected op-pkgs list handler to be called")
	}
}

func TestOpPkgDownloadCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"op-pkgs.download": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"op-pkgs", "download"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected op-pkgs download handler to be called")
	}
}

func TestMetricsCommandWithoutSubcommandShowsHelp(t *testing.T) {
	cmd := newRootCommandWithApp(App{})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"metrics"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	help := buf.String()
	for _, expected := range []string{"query", "compile"} {
		if !strings.Contains(help, expected) {
			t.Fatalf("expected metrics help to include %q, got:\n%s", expected, help)
		}
	}
}
