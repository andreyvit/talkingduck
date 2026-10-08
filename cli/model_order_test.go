package cli

import (
	"slices"
	"testing"
)

func TestSharedPresentationOrder(t *testing.T) {
	source := []string{"folder", "agent", "ticket", "revision", "stage", "uber", "tasks", "progress", "input", "questions", "tracked", "untracked", "markdown", "behind", "ahead", "title"}
	want := []string{"agent", "folder", "ticket", "stage", "uber", "progress", "input", "revision", "tasks", "questions", "tracked", "behind", "ahead", "title"}
	for _, legacy := range []bool{false, true} {
		raw, expected := slices.Clone(source), slices.Clone(want)
		if legacy {
			raw = slices.Delete(raw, 5, 6)
			expected = slices.Delete(expected, 4, 5)
		}
		model := modelTable(TableView{Rows: [][]string{raw}, WorktreePaths: []string{"/folder"}, SpecPaths: []string{""}, AgentLinks: []string{""}})
		for i, col := range model.Columns {
			value := expected[i]
			if col.ID == "tracked" {
				value = "?"
			}
			if col.ID != expected[i] || model.Rows[0].Cells[i].Text() != value {
				t.Fatalf("legacy=%v col=%+v cell=%q", legacy, col, model.Rows[0].Cells[i].Text())
			}
		}
	}
}

func TestMissingAgentIsBlank(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{"--", ""}, {"?", "?"}, {"Grok", "Grok"}} {
		row := make([]string, 16)
		row[1] = tc.raw
		model := modelTable(TableView{Rows: [][]string{row}, WorktreePaths: []string{"/repo"}, SpecPaths: []string{""}, AgentLinks: []string{""}})
		if got := model.Rows[0].Cells[0].Text(); got != tc.want {
			t.Fatalf("%q: got %q", tc.raw, got)
		}
	}
}

func TestQuestionBadges(t *testing.T) {
	for _, count := range []string{"0", "?", "2"} {
		row := make([]string, 16)
		row[9] = count
		model := modelTable(TableView{Rows: [][]string{row}, WorktreePaths: []string{"/repo"}, SpecPaths: []string{""}, AgentLinks: []string{""}})
		for i, col := range model.Columns {
			if col.ID == "questions" {
				item := model.Rows[0].Cells[i].Lines[0].Items[0]
				if col.Label != "Q/S" || col.Align != "right" || item.Badge != (count == "2") {
					t.Fatal(col, item)
				}
				if count == "2" && item.Color != "orange" {
					t.Fatal(item)
				}
			}
		}
	}
}
