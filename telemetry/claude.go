package telemetry

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Claude owns its login and refresh. Never read its Keychain from TalkingDuck.
func collectClaude(ctx context.Context, p Provider) Provider {
	home, _ := os.UserHomeDir()
	var binary string
	for _, candidate := range []string{filepath.Join(home, ".local/bin/claude"), "/opt/homebrew/bin/claude", "/usr/local/bin/claude", "claude"} {
		if found, err := exec.LookPath(candidate); err == nil {
			binary = found
			break
		}
	}
	if binary == "" {
		p.Error = "Claude CLI not installed"
		return p
	}
	dir, err := os.MkdirTemp("", "talkingduck-claude-usage-")
	if err != nil {
		p.Error = "Cannot prepare Claude usage check"
		return p
	}
	defer os.RemoveAll(dir)
	raw, err := runClaudeUsage(ctx, binary, dir)
	if err != nil {
		p.State = "error"
		p.Error = "Claude CLI usage check failed"
		return p
	}
	value, reset, err := parseClaudeUsage(string(raw), time.Now())
	if err != nil {
		p.Error = "Claude CLI weekly usage unavailable"
		return p
	}
	p.State = "ok"
	p.LoggedIn = true
	p.RemainingPercent = &value
	p.ResetsAt = reset
	p.ObservedAt = time.Now()
	// CLI output has no stable account identifier. Do not carry readings across failed probes.
	return p
}

func runClaudeUsage(parent context.Context, binary, dir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "--print", "/usage", "--no-session-persistence", "--allowed-tools", "", "--strict-mcp-config", "--settings", `{"remoteControlAtStartup":false,"disableAllHooks":true}`)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	var out limitedUsageOutput
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.data, nil
}

type limitedUsageOutput struct{ data []byte }

func (b *limitedUsageOutput) Write(p []byte) (int, error) {
	if len(b.data)+len(p) > 64<<10 {
		return 0, fmt.Errorf("usage output too large")
	}
	b.data = append(b.data, p...)
	return len(p), nil
}

var claudeWeek = regexp.MustCompile(`(?m)^Current week \(all models\): ([0-9]+(?:\.[0-9]+)?)% used · resets (.+) \(([^()]+)\)\s*$`)

func parseClaudeUsage(raw string, now time.Time) (float64, *time.Time, error) {
	matches := claudeWeek.FindAllStringSubmatch(raw, -1)
	if len(matches) != 1 {
		return 0, nil, fmt.Errorf("missing or ambiguous weekly usage")
	}
	m := matches[0]
	used, err := strconv.ParseFloat(m[1], 64)
	if err != nil || used < 0 || used > 100 {
		return 0, nil, fmt.Errorf("invalid weekly percentage")
	}
	loc, err := time.LoadLocation(m[3])
	if err != nil {
		return 0, nil, fmt.Errorf("unknown reset timezone")
	}
	localNow := now.In(loc)
	// Printed dates omit the year; accept only an upcoming weekly reset.
	for _, year := range []int{localNow.Year(), localNow.Year() + 1} {
		for _, layout := range []string{"2006 Jan 2 at 3:04pm", "2006 Jan 2 at 3pm"} {
			reset, e := time.ParseInLocation(layout, fmt.Sprintf("%d %s", year, strings.TrimSpace(m[2])), loc)
			if e == nil && reset.After(now) && reset.Sub(now) <= 8*24*time.Hour {
				return 100 - used, &reset, nil
			}
		}
	}
	return 0, nil, fmt.Errorf("invalid weekly reset")
}
