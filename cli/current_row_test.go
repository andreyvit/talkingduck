package cli

import (
	"strings"
	"testing"
)

func TestHistoryAddsToCurrentRows(t *testing.T) {
	for _, state := range []string{"CLEAN", "DIRTY", "MARKDN"} {
		t.Run(state, func(t *testing.T) {
			historical := map[string]any{"path": ".spec/old.md", "display_title": "Old task", "workflow": map[string]any{"stage": "PUSHED"}, "primary_session": map[string]any{"engine": "claude", "id": "old-chat"}}
			repo := map[string]any{"path": "/example/repo", "git": map[string]any{"cleanliness": state}, "specs": []any{historical}}
			data := map[string]any{"repositories": []any{repo}}
			view, err := BuildTable(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(view.Rows) != 2 || view.Rows[0][len(view.Rows[0])-1] != state || view.Rows[0][0] != "repo" || view.Rows[1][0] != "" || view.Rows[1][4] != "PUSHED" {
				t.Fatal(view.Rows)
			}
			if len(view.WorktreePaths) != 2 || view.WorktreePaths[1] != "/example/repo" || len(view.SpecPaths) != 2 || view.SpecPaths[0] != "" || view.SpecPaths[1] != "/example/repo/.spec/old.md" || len(view.AgentLinks) != 2 || view.AgentLinks[0] != "" {
				t.Fatal(view)
			}
			ai, err := FormatAI(data)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(ai, "\n")
			if strings.Contains(lines[0], "spec=") || !strings.Contains(lines[1], "spec=.spec/old.md") || !strings.Contains(lines[1], "session=old-chat") {
				t.Fatal(ai)
			}
			repo["specs"] = []any{map[string]any{"workflow": map[string]any{"stage": "EX"}}, historical}
			view, err = BuildTable(data)
			if err != nil || len(view.Rows) != 2 || view.Rows[0][4] != "EX" {
				t.Fatal(view, err)
			}
		})
	}
}
