package cli

import (
	"flag"
	"io"
	"strings"
	"testing"
)

func TestAIOutput(t *testing.T) {
	task := taskView{Path: ".spec/example.md", Tickets: []string{"DEV-1234", "DEV-5678"}, DisplayTitle: "Long title stays on one line", Workflow: workflowView{Round: 1, Stage: "HL", SubtasksDone: 0, SubtasksTotal: 0, Progress: "✓", Questions: 0, Suggestions: 2}}
	task.PrimarySession.Engine = "grok"
	task.PrimarySession.ID = "session-example"
	g := gitView{Tracked: 229, Untracked: 5, Informational: 4, Behind: 27, Ahead: 0, Cleanliness: "DIRTY"}
	report := tableReport{Repositories: []worktreeView{{Path: "/work/back2", Git: g, Specs: []taskView{task}}, {Path: "/work/back3", Git: gitView{Tracked: 0, Untracked: 0, Informational: 0, Behind: 0, Ahead: 0, Cleanliness: "CLEAN"}}}}
	got := renderAI(report)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 || lines[2] != aiKey {
		t.Fatal(got)
	}
	want := "/work/back2 Grok DEV-1234,DEV-5678 1 HL 0/0 ✓ - 0/2 ±229 ?5 ℹ︎4 ⇣27 ⇡0 Long title stays on one line spec=.spec/example.md session=session-example"
	if lines[0] != want || !strings.HasSuffix(lines[1], "CLEAN") {
		t.Fatal(got)
	}
	if strings.Contains(got, "\x1b[") || strings.Contains(got, "  ") {
		t.Fatal("color or alignment padding")
	}
}

func TestJSONFlagRemoved(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	RegisterFlags(fs)
	if err := fs.Parse([]string{"--json"}); err == nil {
		t.Fatal("legacy JSON flag accepted")
	}
}

func TestSessionListIsPlainText(t *testing.T) {
	got, err := renderSessions(map[string]any{"sessions": []any{map[string]any{"engine": "grok", "id": "example-id", "cwd": "/work/back2", "title": "Example", "activity": map[string]any{"value": "unknown"}, "question": map[string]any{"value": "yes"}}}, "sources": []any{map[string]any{"engine": "codex", "status": "partial", "warnings": []string{"schema unavailable"}}}})
	if err != nil || !strings.Contains(got, "session=example-id") || !strings.Contains(got, "schema unavailable") || !strings.HasPrefix(strings.Split(got, "\n")[2], "Key:") {
		t.Fatal(got, err)
	}
}
