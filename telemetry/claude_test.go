package telemetry

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudePrintedWeeklyUsage(t *testing.T) {
	now := time.Date(2026, 10, 21, 7, 0, 0, 0, time.UTC)
	raw := "Current session: 80% used · resets Oct 21 at 5:20pm (Indian/Maldives)\nCurrent week (all models): 23% used · resets Oct 26 at 11am (Indian/Maldives)\nCurrent week (Fable): 90% used · resets Oct 26 at 11am (Indian/Maldives)\n"
	value, reset, err := parseClaudeUsage(raw, now)
	if err != nil || value != 77 || !reset.Equal(time.Date(2026, 10, 26, 6, 0, 0, 0, time.UTC)) {
		t.Fatalf("%v %v %v", value, reset, err)
	}
	for _, bad := range []string{strings.ReplaceAll(raw, "all models", "Sonnet"), raw + raw, strings.ReplaceAll(raw, "23%", "123%"), strings.ReplaceAll(raw, "Oct 26", "Oct 1"), strings.ReplaceAll(raw, "Indian/Maldives", "Unknown/Zone"), "Please log in", "Current session: 0% used"} {
		if _, _, err := parseClaudeUsage(bad, now); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
	_, reset, err = parseClaudeUsage("Current week (all models): 0% used · resets Jan 2 at 3:30pm (UTC)", time.Date(2026, 12, 30, 0, 0, 0, 0, time.UTC))
	if err != nil || reset.Year() != 2027 {
		t.Fatalf("year rollover: %v %v", reset, err)
	}
}
func TestClaudeUsageProcessIsolation(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "claude")
	script := `#!/bin/sh
printf '%s\n' "$@"
pwd
`
	if err := os.WriteFile(binary, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	out, err := runClaudeUsage(context.Background(), binary, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, arg := range []string{"--print\n/usage\n", "--no-session-persistence", "--allowed-tools\n\n", "--strict-mcp-config", `"disableAllHooks":true`, `"remoteControlAtStartup":false`, dir} {
		if !strings.Contains(string(out), arg) {
			t.Errorf("missing %q", arg)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runClaudeUsage(ctx, binary, dir); err == nil {
		t.Fatal("ignored cancellation")
	}
	var output limitedUsageOutput
	if _, err := output.Write(make([]byte, 65537)); err == nil {
		t.Fatal("unbounded output")
	}
}
