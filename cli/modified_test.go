package cli

import "testing"

func TestModifiedCount(t *testing.T) {
	for _, tc := range [][4]string{{"±4", "?5", "ℹ︎2", "±11"}, {"±0", "?0", "ℹ︎0", "±0"}, {"±0", "?0", "ℹ︎3", ".md"}, {"±?", "?5", "ℹ︎0", "±?"}, {"ERROR", "", "", "ERROR"}, {"", "", "", ""}} {
		if got := modifiedCount(tc[0], tc[1], tc[2]); got != tc[3] {
			t.Fatal(tc, got)
		}
	}
}
func TestMergedModifiedColumn(t *testing.T) {
	row := make([]string, 16)
	row[10], row[11], row[12] = "±4", "?5", "ℹ︎2"
	model := modelTable(TableView{Rows: [][]string{row}, WorktreePaths: []string{"/repo"}, SpecPaths: []string{""}, AgentLinks: []string{""}})
	found := false
	for i, col := range model.Columns {
		if col.ID == "untracked" || col.ID == "markdown" {
			t.Fatal("New column remains")
		}
		if col.ID == "tracked" {
			found = true
			if model.Rows[0].Cells[i].Text() != "11" {
				t.Fatal(model.Rows[0].Cells[i])
			}
		}
	}
	if !found {
		t.Fatal("missing Mod")
	}
}
