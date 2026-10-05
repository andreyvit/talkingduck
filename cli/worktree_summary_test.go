package cli

import (
	"strings"
	"testing"
)

func TestTasklessUnpushedWorkStaysVisible(t *testing.T) {
	data := map[string]any{"repositories": []any{map[string]any{
		"path": "/example/repo", "git": map[string]any{"cleanliness": "CLEAN", "ahead": 2},
		"unassigned_commits": []string{"one", "two"}, "specs": []any{},
	}}}
	table, err := BuildTable(data)
	if err != nil {
		t.Fatal(err)
	}
	row := table.Rows[0]
	if got := row[len(row)-1]; got != "2 unpushed commits; spec unknown" {
		t.Fatal(got)
	}
	ai, err := FormatAI(data)
	if err != nil || !strings.Contains(ai, "2 unpushed commits; spec unknown") {
		t.Fatal(ai, err)
	}
	if got := worktreeSummary(worktreeView{Git: gitView{Cleanliness: "CLEAN"}}); got != "" {
		t.Fatal(got)
	}
}
