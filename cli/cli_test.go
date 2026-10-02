package cli

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSharedFlagsAndExitCodes(t *testing.T) {
	fs := flag.NewFlagSet("desktop", flag.ContinueOnError)
	opts := RegisterFlags(fs)
	gui := fs.Bool("gui", false, "GUI")
	missing := filepath.Join(t.TempDir(), "missing")
	if err := fs.Parse([]string{"--gui", "-no-agents", "-ai", "-color", "always", missing}); err != nil {
		t.Fatal(err)
	}
	if !*gui {
		t.Fatal("entry point flag not parsed")
	}
	var out, stderr bytes.Buffer
	if code := RunTo(opts, fs.Args(), &out, &stderr); code != 1 {
		t.Fatal(code)
	}
	if !strings.Contains(out.String(), "ERROR") || strings.Contains(out.String(), "\x1b[") || !strings.HasSuffix(strings.TrimSpace(out.String()), aiKey) {
		t.Fatal(out.String())
	}
	opts.Details = true
	out.Reset()
	if code := RunTo(opts, fs.Args(), &out, &stderr); code != 1 || !strings.HasSuffix(strings.TrimSpace(out.String()), aiKey) {
		t.Fatal(code, out.String())
	}
	opts.Color = "invalid"
	out.Reset()
	stderr.Reset()
	if code := RunTo(opts, nil, &out, &stderr); code != 2 || out.Len() != 0 || !strings.Contains(stderr.String(), "-color") {
		t.Fatal(code, stderr.String())
	}
	opts.Color = "auto"
	opts.Inventory.AgentLimit = -1
	if code := RunTo(opts, nil, &out, &stderr); code != 2 {
		t.Fatal(code)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestRunSuccessAndOutputErrors(t *testing.T) {
	root := t.TempDir()
	if b, e := exec.Command("git", "init", "-q", root).CombinedOutput(); e != nil {
		t.Fatal(string(b), e)
	}
	if b, e := exec.Command("git", "-C", root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Initial").CombinedOutput(); e != nil {
		t.Fatal(string(b), e)
	}
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	opts := RegisterFlags(fs)
	opts.Inventory.NoAgents = true
	var out, stderr bytes.Buffer
	if code := RunTo(opts, []string{root}, &out, &stderr); code != 0 {
		t.Fatal(code, stderr.String())
	}
	if !strings.Contains(out.String(), "CLEAN") || strings.Contains(out.String(), "\x1b[") {
		t.Fatal(out.String())
	}
	for _, mode := range []string{"table", "ai", "details"} {
		opts.AI = mode == "ai"
		opts.Details = mode == "details"
		stderr.Reset()
		if code := RunTo(opts, []string{root}, brokenWriter{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "output unavailable") {
			t.Fatal(mode, code, stderr.String())
		}
	}
}

func TestMultilineTableAndUnicode(t *testing.T) {
	task := taskView{Tickets: []string{"DEV-1234", "DEV-5678"}, DisplayTitle: "example", Workflow: workflowView{Stage: "HL", Round: 1, SubtasksDone: 0, SubtasksTotal: 0, Progress: "?", Interaction: "?", Suggestions: 0}}
	g := gitView{Tracked: 0, Untracked: 0, Informational: 1, Ahead: 0, Behind: 0, Cleanliness: "CLEAN"}
	report := tableReport{Repositories: []worktreeView{{Path: "/back1", Git: g, Specs: []taskView{task}}, {Path: "/back2", Git: g}}}
	plain := render(report, 120, false)
	if regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(render(report, 120, true), "") != plain {
		t.Fatal("colors changed layout")
	}
	lines := strings.Split(plain, "\n")
	if strings.TrimSpace(lines[1]) != "DEV-5678" || !strings.Contains(lines[2], "CLEAN") {
		t.Fatal(lines)
	}
	for _, line := range lines {
		if cellWidth(line) > 120 {
			t.Fatal(line)
		}
	}
	if cellWidth("ℹ︎界e\u0301") != 4 {
		t.Fatal("Unicode width")
	}
	t.Setenv("COLUMNS", "150")
	if tableWidth(0, io.Discard) != 150 || tableWidth(90, io.Discard) != 90 {
		t.Fatal("width selection")
	}
}
