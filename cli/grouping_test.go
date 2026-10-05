package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestWorktreeGroupingRetainsPathsAndAIIdentity(t *testing.T) {
	data := map[string]any{"repositories": []any{
		map[string]any{"path": "/one/back2", "specs": []any{map[string]any{"path": ".spec/a.md"}, map[string]any{"path": ".spec/b.md"}}},
		map[string]any{"path": "/two/back2"},
	}}
	view, err := BuildTable(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{view.Rows[0][0], view.Rows[1][0], view.Rows[2][0]}; !reflect.DeepEqual(got, []string{"back2", "", "back2"}) {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(view.WorktreePaths, []string{"/one/back2", "/one/back2", "/two/back2"}) {
		t.Fatal(view.WorktreePaths)
	}
	report, _ := tableData(data)
	if n := strings.Count(renderAI(report), "/one/back2 "); n != 2 {
		t.Fatalf("AI lost full worktree identity: %d", n)
	}
}
func TestPushedRowUsesOneColor(t *testing.T) {
	report := tableReport{Repositories: []worktreeView{{Path: "/repo", Specs: []taskView{{DisplayTitle: "Historical title", Workflow: workflowView{Stage: "PUSHED"}}}}}}
	out := strings.Split(render(report, 200, true), "\n")[1]
	if !strings.Contains(out, paint("Historical title", 245)) {
		t.Fatalf("title not dimmed: %q", out)
	}
	// Every colored cell must use the same muted foreground, including title/agent.
	for _, part := range strings.Split(out, "\x1b[38;5;")[1:] {
		if !strings.HasPrefix(part, "245m") {
			t.Fatalf("non-muted pushed cell: %q", part)
		}
	}
}
