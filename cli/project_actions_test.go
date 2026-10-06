package cli

import (
	"github.com/gdamore/tcell/v2"
	"talkingduck/table"
	"testing"
)

func TestProjectActionsOnGroupedRows(t *testing.T) {
	view := TableView{Rows: [][]string{make([]string, 16), make([]string, 16)}, WorktreePaths: []string{"/work/repo", "/work/repo"}, SpecPaths: []string{"", "/work/repo/.spec/task.md"}, AgentLinks: []string{"", ""}}
	model := modelTable(view)
	s := &tuiState{model: model, selected: 1}
	var kinds []string
	s.perform = func(_ string, item table.Item) {
		if item.Link.Target != "/work/repo" {
			t.Fatal(item.Link)
		}
		kinds = append(kinds, item.Link.Kind)
	}
	for _, key := range "eg" {
		s.key(tcell.NewEventKey(tcell.KeyRune, key, tcell.ModNone))
	}
	if len(kinds) != 2 || kinds[0] != "editor" || kinds[1] != "git" {
		t.Fatal(kinds)
	}
}
