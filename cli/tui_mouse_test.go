package cli

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"talkingduck/table"
	"testing"
)

func TestTUIMouseActionsAndHover(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(56, 12)
	s := tuiState{model: tuiFixture()}
	var calls []string
	s.perform = func(v string, i table.Item) { calls = append(calls, v+":"+i.Text) }
	s.draw(screen)
	mouse := func(x, y int, b tcell.ButtonMask) {
		s.mouse(tcell.NewEventMouse(x, y, b, tcell.ModNone))
		s.draw(screen)
	}
	click := func(x, y int) { mouse(x, y, tcell.Button1); mouse(x, y, tcell.ButtonNone) }
	// Folder copy occupies x=4; it appears on hover without moving the text.
	r, _, _, _ := screen.GetContent(4, 2)
	if r == '©' {
		t.Fatal("copy visible without hover")
	}
	mouse(6, 2, tcell.ButtonNone) // Whitespace within the single-item folder cell.
	r, _, _, _ = screen.GetContent(4, 2)
	if r != '©' {
		t.Fatal("copy not visible on hover")
	}
	click(4, 2)
	click(8, 3) // Second ticket, separate physical line.
	if fmt.Sprint(calls) != "[c:one o:DEV-2]" {
		t.Fatal(calls)
	}
	mouse(8, 2, tcell.Button1)
	mouse(9, 2, tcell.Button1) // Held motion never opens repeatedly.
	if len(calls) != 2 {
		t.Fatal(calls)
	}
	mouse(1, 4, tcell.ButtonNone) // Release over another row cancels.
	if len(calls) != 2 {
		t.Fatal(calls)
	}
	mouse(8, 2, tcell.Button1)
	s.replace(tuiFixture())
	s.draw(screen)
	mouse(8, 2, tcell.ButtonNone)
	if len(calls) != 2 {
		t.Fatal("stale press survived refresh", calls)
	}
	click(0, 4)
	if s.selected != 1 || len(calls) != 2 {
		t.Fatal("row marker acted as a link", s.selected, calls)
	}
	// Modal chooser cannot click through to the table.
	s.choices = s.model.Rows[0].Matches("c", "t")
	s.verb = "c"
	s.draw(screen)
	click(1, 2)
	if len(calls) != 2 {
		t.Fatal("chooser click through")
	}
	click(4, 6)
	if fmt.Sprint(calls) != "[c:one o:DEV-2 c:DEV-2]" {
		t.Fatal(calls)
	}
	screen.SetSize(12, 4)
	s.draw(screen)
	click(4, 2)
	if len(calls) != 3 {
		t.Fatal("small screen retained targets")
	}
}

func TestTUIMouseScrollDoesNotSnapToSelection(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(56, 12)
	model := tuiFixture()
	for n := 0; n < 15; n++ {
		row := model.Rows[1]
		row.ID = fmt.Sprint(n)
		model.Rows = append(model.Rows, row)
	}
	s := tuiState{model: model}
	s.draw(screen)
	s.mouse(tcell.NewEventMouse(20, 3, tcell.WheelDown, tcell.ModNone))
	s.draw(screen)
	if s.top != 3 || s.selected != 0 {
		t.Fatal(s.top, s.selected)
	}
	s.key(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	s.draw(screen)
	if s.top != 2 || s.selected != 1 {
		t.Fatal("keyboard did not reveal selection", s.top, s.selected)
	}
}
