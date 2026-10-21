package telemetry

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUsageParsing(t *testing.T) {
	tests := []struct {
		id, body string
		want     float64
		valid    bool
	}{
		{"codex", `{"rate_limit":{"primary_window":{"used_percent":91,"limit_window_seconds":18000},"secondary_window":{"used_percent":23,"limit_window_seconds":604800,"reset_at":1900000000}}}`, 77, true},
		{"codex", `{"rate_limit":{"secondary_window":{"used_percent":23,"limit_window_seconds":18000}}}`, 0, false},
		{"claude", `{"seven_day":{"utilization":81.5,"resets_at":"2026-10-25T01:02:03Z"}}`, 18.5, true},
		{"claude", `{"seven_day":{"utilization":null}}`, 0, false},
		{"claude", `{"seven_day":{"utilization":-1}}`, 0, false},
		{"grok", `{"config":{"creditUsagePercent":30,"currentPeriod":{"start":"2026-10-20T00:00:00Z","end":"2026-10-27T00:00:00Z"}}}`, 70, true},
		{"grok", `{"config":{"creditUsagePercent":30,"currentPeriod":{"start":"2026-10-01T00:00:00Z","end":"2026-11-01T00:00:00Z"}}}`, 0, false},
		{"grok", `{"config":{"currentPeriod":{"start":"2026-10-20T00:00:00Z","end":"2026-10-27T00:00:00Z"}}}`, 0, false},
		{"grok", `{"config":{"onDemandCap":{"val":"100"},"onDemandUsed":{"val":"23"},"currentPeriod":{"start":"2026-10-20T00:00:00Z","end":"2026-10-27T00:00:00Z"}}}`, 77, true},
		{"grok", `{"config":{"onDemandCap":{"val":"0"},"onDemandUsed":{"val":"0"},"currentPeriod":{"start":"2026-10-20T00:00:00Z","end":"2026-10-27T00:00:00Z"}}}`, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.id+tt.body, func(t *testing.T) {
			got, _, err := parseUsage(tt.id, []byte(tt.body))
			if (err == nil) != tt.valid || tt.valid && got != tt.want {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}
func TestStaleAccountIsolation(t *testing.T) {
	n := 42.
	now := time.Now()
	old := Provider{identity: "one", RemainingPercent: &n, ObservedAt: now}
	p := merge(old, Provider{identity: "one", State: "error"})
	if p.State != "stale" || p.RemainingPercent == nil || !p.ObservedAt.Equal(now) {
		t.Fatal("lost stale reading")
	}
	p = merge(old, Provider{identity: "two", State: "error"})
	if p.RemainingPercent != nil {
		t.Fatal("leaked other account")
	}
	p = merge(old, Provider{identity: "one", State: "auth"})
	if p.RemainingPercent != nil {
		t.Fatal("kept invalid auth")
	}
}
func TestManagerOffAndGeneration(t *testing.T) {
	c := Config{Codex: "off", Claude: "off", Grok: "off"}
	m := New(c, nil)
	defer m.Close()
	m.Update(Config{PingTarget: "1.1.1.1", Codex: "off", Claude: "off", Grok: "off"})
	m.publish(1, func(s *Snapshot) { s.Ping.State = "wrong" })
	s := m.Snapshot()
	if s.Ping.State != "off" || len(s.Providers) != 3 {
		t.Fatal(s)
	}
	s.Providers[0].State = "wrong"
	if m.Snapshot().Providers[0].State != "off" {
		t.Fatal("mutable snapshot")
	}
}

func TestRetryDelay(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	if retryDelay("120", now) != 2*time.Minute {
		t.Fatal("seconds")
	}
	if retryDelay(now.Add(time.Hour).Format(http.TimeFormat), now) != time.Hour {
		t.Fatal("date")
	}
	if retryDelay("bogus", now) != 0 {
		t.Fatal("invalid")
	}
}

func TestRecentActivity(t *testing.T) {
	root := t.TempDir()
	now := time.Now()
	cutoff := now.Add(-14 * 24 * time.Hour)
	if recentFiles(root, "codex", cutoff) {
		t.Fatal("empty root")
	}
	file := filepath.Join(root, "rollout.jsonl")
	if err := os.WriteFile(file, []byte("not parsed"), 0600); err != nil {
		t.Fatal(err)
	}
	if !recentFiles(root, "codex", cutoff) {
		t.Fatal("recent session")
	}
	old := cutoff.Add(-time.Hour)
	os.Chtimes(file, old, old)
	if recentFiles(root, "codex", cutoff) {
		t.Fatal("old session")
	}
	os.WriteFile(filepath.Join(root, "auth.json"), []byte("ignored"), 0600)
	if recentFiles(root, "codex", cutoff) {
		t.Fatal("credentials are not activity")
	}
	os.Symlink(file, filepath.Join(root, "linked.jsonl"))
	if recentFiles(root, "codex", cutoff) {
		t.Fatal("symlink is not activity")
	}
}
