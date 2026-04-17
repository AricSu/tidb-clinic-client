package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootHelpOnlyShowsCapabilityFirstCommands(t *testing.T) {
	cmd := newRootCommandWithApp(App{})
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	help := buf.String()
	for _, name := range []string{"clusters", "metrics", "logs", "sql", "configs", "profiling", "diagnostics", "capabilities"} {
		if !strings.Contains(help, "  "+name) {
			t.Fatalf("expected help to include %q, got:\n%s", name, help)
		}
	}
	for _, removed := range []string{"  cloud"} {
		if strings.Contains(help, removed) {
			t.Fatalf("expected help to omit %q, got:\n%s", strings.TrimSpace(removed), help)
		}
	}
}

func TestLogsSearchCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"logs.search": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"logs", "search"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected logs search handler to be called")
	}
}

func TestSQLSlowQueryRecordsCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"sql.slowquery-records": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"sql", "slowquery-records"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected slowquery-records handler to be called")
	}
}

func TestConfigsGetCommandRoutesToInjectedHandler(t *testing.T) {
	called := false
	cmd := newRootCommandWithApp(App{
		"configs.get": func() error {
			called = true
			return nil
		},
	})
	cmd.SetArgs([]string{"configs", "get"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !called {
		t.Fatalf("expected configs get handler to be called")
	}
}
