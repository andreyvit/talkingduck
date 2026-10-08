package cli

import "testing"

func TestModifiedCount(t *testing.T) {
	for _, tc := range [][3]string{{"±4", "?5", "±9"}, {"±0", "?0", "±0"}, {"±?", "?5", "±?"}, {"ERROR", "", "ERROR"}, {"", "", ""}} {
		if got := modifiedCount(tc[0], tc[1]); got != tc[2] {
			t.Fatal(tc, got)
		}
	}
}
func TestMergedModifiedColumn(t *testing.T) {
	row := make([]string, 16)
	row[10], row[11] = "±4", "?5"
	model := modelTable(TableView{Rows: [][]string{row}, WorktreePaths: []string{"/repo"}, SpecPaths: []string{""}, AgentLinks: []string{""}})
	found := false
	for i, col := range model.Columns {
		if col.ID == "untracked" {
			t.Fatal("New column remains")
		}
		if col.ID == "tracked" {
			found = true
			if model.Rows[0].Cells[i].Text() != "9" {
				t.Fatal(model.Rows[0].Cells[i])
			}
		}
	}
	if !found {
		t.Fatal("missing Mod")
	}
}
