package cli

import (
	"slices"
	"testing"
)

func TestSharedPresentationOrder(t *testing.T) {
	source := []string{"folder", "agent", "ticket", "revision", "stage", "uber", "tasks", "progress", "input", "questions", "tracked", "untracked", "markdown", "behind", "ahead", "title"}
	want := []string{"folder", "agent", "ticket", "stage", "uber", "progress", "input", "revision", "tasks", "questions", "tracked", "untracked", "markdown", "behind", "ahead", "title"}
	for _, legacy := range []bool{false, true} {
		raw, expected := slices.Clone(source), slices.Clone(want)
		if legacy {
			raw = slices.Delete(raw, 5, 6)
			expected = slices.Delete(expected, 4, 5)
		}
		model := modelTable(TableView{Rows: [][]string{raw}, WorktreePaths: []string{"/folder"}, SpecPaths: []string{""}, AgentLinks: []string{""}})
		for i, col := range model.Columns {
			if col.ID != expected[i] || model.Rows[0].Cells[i].Text() != expected[i] {
				t.Fatalf("legacy=%v col=%+v cell=%q", legacy, col, model.Rows[0].Cells[i].Text())
			}
		}
	}
}
