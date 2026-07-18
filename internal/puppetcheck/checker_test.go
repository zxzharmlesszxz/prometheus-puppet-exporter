package puppetcheck

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckerReadsFixtureState(t *testing.T) {
	t.Parallel()

	fixtures := filepath.Join("..", "..", "testdata")
	checker := NewChecker(Paths{
		Config:  filepath.Join(fixtures, "puppet.conf"),
		Lock:    filepath.Join(fixtures, "agent_disabled.lock"),
		Summary: filepath.Join(fixtures, "last_run_summary.yaml"),
	})

	snapshot := checker.Snapshot(context.Background(), time.Unix(1774433244, 0))
	if !snapshot.Success {
		t.Fatal("snapshot.Success = false, want true")
	}
	if !snapshot.Config.Up || !snapshot.Config.Valid {
		t.Fatalf("Config Up/Valid = %v/%v, want true/true", snapshot.Config.Up, snapshot.Config.Valid)
	}
	if snapshot.Config.Server != "puppet.example.net" {
		t.Fatalf("Config.Server = %q, want puppet.example.net", snapshot.Config.Server)
	}
	if !snapshot.Lock.Up {
		t.Fatal("Lock.Up = false, want true")
	}
	if !snapshot.Lock.Valid {
		t.Fatal("Lock.Valid = false, want true")
	}
	const wantDisabledMessage = "Very long test disable message for grafana dashboard testing"
	if snapshot.Lock.DisabledMessage != wantDisabledMessage {
		t.Fatalf("Lock.DisabledMessage = %q, want %q", snapshot.Lock.DisabledMessage, wantDisabledMessage)
	}
	if !snapshot.Summary.Up {
		t.Fatal("Summary.Up = false, want true")
	}
	if !snapshot.Summary.Valid {
		t.Fatal("Summary.Valid = false, want true")
	}
	if snapshot.Summary.RunSuccess != 1 {
		t.Fatalf("Summary.RunSuccess = %v, want 1", snapshot.Summary.RunSuccess)
	}
	if snapshot.Summary.LastRunTimestamp != 1791596213 {
		t.Fatalf("Summary.LastRunTimestamp = %v, want 1791596213", snapshot.Summary.LastRunTimestamp)
	}
	if _, ok := snapshot.Summary.Durations["last_run"]; ok {
		t.Fatal("Summary.Durations retains last_run, want it moved to LastRunTimestamp")
	}
}

func TestCheckerTreatsMissingLockAsEnabledAgent(t *testing.T) {
	t.Parallel()

	fixtures := filepath.Join("..", "..", "testdata")
	checker := NewChecker(Paths{
		Config:  filepath.Join(fixtures, "puppet.conf"),
		Lock:    filepath.Join(t.TempDir(), "missing-agent_disabled.lock"),
		Summary: filepath.Join(fixtures, "last_run_summary.yaml"),
	})

	snapshot := checker.Snapshot(context.Background(), time.Unix(1774433244, 0))
	if !snapshot.Success {
		t.Fatal("snapshot.Success = false, want true")
	}
	if !snapshot.Lock.Up {
		t.Fatal("Lock.Up = false, want true")
	}
	if !snapshot.Lock.Valid {
		t.Fatal("Lock.Valid = false, want true")
	}
	if snapshot.Lock.DisabledMessage != "" {
		t.Fatalf("Lock.DisabledMessage = %q, want empty", snapshot.Lock.DisabledMessage)
	}
	if snapshot.Lock.ReadErrorsTotal != 0 || snapshot.Lock.ParseErrorsTotal != 0 {
		t.Fatalf("Lock counters = read %d parse %d, want 0/0", snapshot.Lock.ReadErrorsTotal, snapshot.Lock.ParseErrorsTotal)
	}
}

func TestCheckerTracksSourceFailures(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "agent_disabled.lock")
	summaryPath := filepath.Join(dir, "last_run_summary.yaml")
	if err := os.WriteFile(lockPath, []byte(`{"disabled_message":`), 0o600); err != nil {
		t.Fatalf("WriteFile(lock) error = %v", err)
	}
	if err := os.WriteFile(summaryPath, []byte("events: [broken"), 0o600); err != nil {
		t.Fatalf("WriteFile(summary) error = %v", err)
	}

	checker := NewChecker(Paths{
		Config:  filepath.Join(dir, "missing-puppet.conf"),
		Lock:    lockPath,
		Summary: summaryPath,
	})

	snapshot := checker.Snapshot(context.Background(), time.Unix(1774433244, 0))
	if snapshot.Success {
		t.Fatal("snapshot.Success = true, want false")
	}
	if snapshot.Config.Up || snapshot.Config.Err == nil || snapshot.Config.ReadErrorsTotal != 1 {
		t.Fatalf("Config failure = up %v err %v read %d, want false/non-nil/1", snapshot.Config.Up, snapshot.Config.Err, snapshot.Config.ReadErrorsTotal)
	}
	if !snapshot.Lock.Up || snapshot.Lock.Valid || snapshot.Lock.Err == nil || snapshot.Lock.ParseErrorsTotal != 1 {
		t.Fatalf("Lock failure = up %v valid %v err %v parse %d, want true/false/non-nil/1", snapshot.Lock.Up, snapshot.Lock.Valid, snapshot.Lock.Err, snapshot.Lock.ParseErrorsTotal)
	}
	if !snapshot.Summary.Up || snapshot.Summary.Valid || snapshot.Summary.Err == nil || snapshot.Summary.ParseErrorsTotal != 1 {
		t.Fatalf("Summary failure = up %v valid %v err %v parse %d, want true/false/non-nil/1", snapshot.Summary.Up, snapshot.Summary.Valid, snapshot.Summary.Err, snapshot.Summary.ParseErrorsTotal)
	}
}

func TestCheckerHandlesEmptyConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "puppet.conf")
	if err := os.WriteFile(configPath, []byte{}, 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	fixtures := filepath.Join("..", "..", "testdata")
	checker := NewChecker(Paths{
		Config:  configPath,
		Lock:    filepath.Join(fixtures, "agent_disabled.lock"),
		Summary: filepath.Join(fixtures, "last_run_summary.yaml"),
	})

	snapshot := checker.Snapshot(context.Background(), time.Unix(1774433244, 0))
	if !snapshot.Config.Up {
		t.Fatal("Config.Up = false, want true (empty file is parseable)")
	}
	if snapshot.Config.Valid {
		t.Fatal("Config.Valid = true, want false (empty config has no fields)")
	}
	if snapshot.Config.Server != "" || snapshot.Config.Certname != "" || snapshot.Config.Environment != "" {
		t.Fatal("expected empty config fields")
	}
	if !snapshot.Lock.Up || !snapshot.Summary.Up {
		t.Fatal("other sources should still succeed")
	}
	if !snapshot.Lock.Valid || !snapshot.Summary.Valid {
		t.Fatal("other sources should still be valid")
	}
}

func TestCheckerTreatsMalformedConfigAsReadableButInvalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "puppet.conf")
	if err := os.WriteFile(configPath, []byte("[main\nserver = puppet.example.net\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	fixtures := filepath.Join("..", "..", "testdata")
	checker := NewChecker(Paths{
		Config:  configPath,
		Lock:    filepath.Join(fixtures, "agent_disabled.lock"),
		Summary: filepath.Join(fixtures, "last_run_summary.yaml"),
	})

	snapshot := checker.Snapshot(context.Background(), time.Unix(1774433244, 0))
	if !snapshot.Config.Up || snapshot.Config.Valid || snapshot.Config.Err == nil || snapshot.Config.ParseErrorsTotal != 1 {
		t.Fatalf(
			"Config failure = up %v valid %v err %v parse %d, want true/false/non-nil/1",
			snapshot.Config.Up,
			snapshot.Config.Valid,
			snapshot.Config.Err,
			snapshot.Config.ParseErrorsTotal,
		)
	}
}

func TestCheckerRejectsIncompleteSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		summary string
	}{
		{
			name: "missing events failure",
			summary: `events:
  success: 0
  total: 0
resources:
  failed: 0
time:
  last_run: 1774433244
`,
		},
		{
			name: "missing resources failed",
			summary: `events:
  failure: 0
resources:
  total: 1
time:
  last_run: 1774433244
`,
		},
		{
			name: "missing last run timestamp",
			summary: `events:
  failure: 0
resources:
  failed: 0
time:
  total: 1
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			summaryPath := filepath.Join(dir, "last_run_summary.yaml")
			if err := os.WriteFile(summaryPath, []byte(tt.summary), 0o600); err != nil {
				t.Fatalf("WriteFile(summary) error = %v", err)
			}

			fixtures := filepath.Join("..", "..", "testdata")
			checker := NewChecker(Paths{
				Config:  filepath.Join(fixtures, "puppet.conf"),
				Lock:    filepath.Join(fixtures, "agent_disabled.lock"),
				Summary: summaryPath,
			})

			snapshot := checker.Snapshot(context.Background(), time.Unix(1774433244, 0))
			if !snapshot.Summary.Up || snapshot.Summary.Valid || snapshot.Summary.Err == nil || snapshot.Summary.ParseErrorsTotal != 1 {
				t.Fatalf(
					"Summary failure = up %v valid %v err %v parse %d, want true/false/non-nil/1",
					snapshot.Summary.Up,
					snapshot.Summary.Valid,
					snapshot.Summary.Err,
					snapshot.Summary.ParseErrorsTotal,
				)
			}
		})
	}
}
