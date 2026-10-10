package table

import (
	"strings"
	"testing"
)

func TestWorktreeGroupingAndWrapping(t *testing.T) {
	rows := []Row{{ID: "child", Group: "/elsewhere/child", ParentGroup: "/repo"}, {ID: "other", Group: "/other"}, {ID: "parent1", Group: "/repo"}, {ID: "parent2", Group: "/repo"}}
	got := GroupRows(rows)
	for i, want := range []string{"other", "parent1", "parent2", "child"} {
		if got[i].ID != want {
			t.Fatalf("%+v", got)
		}
	}
	copyPath := "/elsewhere/child"
	cell := Cell{Indent: 2, WrapWidth: 14, Small: true, Lines: []Line{{Items: []Item{{Text: "long-worktree-name-with-no-spaces", Chord: "f", Copy: &copyPath}}}}}
	model := Table{Columns: []Column{{ID: "folder", MinWidth: 6}}, Rows: []Row{{Cells: []Cell{cell}}}}
	layout := Format(model, 80, false)
	if layout.Widths[0] > 14 || len(layout.Lines) < 3 {
		t.Fatalf("not bounded/wrapped: %+v", layout)
	}
	text := ""
	for _, line := range layout.Lines {
		for _, span := range line.Spans {
			if span.Text != "" {
				if span.X < 2 {
					t.Fatal("lost indent")
				}
				text += span.Text
			}
		}
	}
	if strings.TrimSpace(text) != cell.Text() {
		t.Fatalf("lost text: %q", text)
	}
	if got := model.Rows[0].Matches("c", "f"); len(got) != 1 || *got[0].Copy != copyPath {
		t.Fatal("copy target lost")
	}
}
