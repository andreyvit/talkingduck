package cli

import (
	"flag"
	"github.com/gdamore/tcell/v2"
	"strings"
	"talkingduck/table"
	"testing"
)

func TestModesMatchAcrossBinaries(t *testing.T) {
	for _, def := range []string{"tui", "gui"} {
		for _, mode := range []string{"", "ai", "status", "tui", "gui"} {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			o := RegisterFlags(fs)
			args := []string{}
			if mode != "" {
				args = append(args, "-"+mode)
			}
			if err := fs.Parse(args); err != nil {
				t.Fatal(err)
			}
			got, err := o.Mode(def)
			want := mode
			if want == "" {
				want = def
			}
			if err != nil || got != want {
				t.Fatal(got, err)
			}
		}
	}
	o := Options{AI: true, TUI: true}
	if _, err := o.Mode("gui"); err == nil {
		t.Fatal("conflicting modes accepted")
	}
}
func tuiFixture() table.Table {
	str := func(s string) *string { return &s }
	cell := func(items ...table.Item) table.Cell { return table.Cell{Lines: []table.Line{{Items: items}}} }
	return table.Table{Columns: []table.Column{{ID: "folder", Label: "Folder", MinWidth: 6}, {ID: "ticket", Label: "Ticket", MinWidth: 8}, {ID: "title", Label: "Title", Flex: true, MinWidth: 8}}, Rows: []table.Row{
		{ID: "one", Group: "/one", State: "running", Cells: []table.Cell{cell(table.Item{Text: "one", Chord: "f", Copy: str("/one"), Link: &table.Action{Kind: "folder", Target: "/one"}}), {Lines: []table.Line{{Items: []table.Item{{Text: "DEV-1", Chord: "t", Copy: str("DEV-1"), Link: &table.Action{Kind: "ticket", Target: "DEV-1"}}}}, {Items: []table.Item{{Text: "DEV-2", Chord: "t", Copy: str("DEV-2"), Link: &table.Action{Kind: "ticket", Target: "DEV-2"}}}}}}, cell(table.Item{Text: "A long title with 界 Unicode"})}},
		{ID: "two", Group: "/one", State: "pushed", Cells: []table.Cell{cell(table.Item{Text: "one", Hidden: true, Chord: "f", Copy: str("/one"), Link: &table.Action{Kind: "folder", Target: "/one"}}), cell(table.Item{Text: "--"}), cell(table.Item{Text: "History"})}},
	}}
}
func TestTUIChordsSelectionAndRefresh(t *testing.T) {
	s := tuiState{model: tuiFixture()}
	refreshed := []string{}
	performed := []string{}
	s.refresh = func(p string) { refreshed = append(refreshed, p) }
	s.perform = func(v string, i table.Item) {
		value := ""
		if v == "c" {
			value = *i.Copy
		} else {
			value = i.Link.Target
		}
		performed = append(performed, v+":"+value)
	}
	key := func(k tcell.Key, r rune) { s.key(tcell.NewEventKey(k, r, tcell.ModNone)) }
	runeKey := func(r rune) { key(tcell.KeyRune, r) }
	runeKey('c')
	runeKey('t')
	if len(s.choices) != 2 {
		t.Fatal(s)
	}
	key(tcell.KeyDown, 0)
	key(tcell.KeyEnter, 0)
	if performed[0] != "c:DEV-2" {
		t.Fatal(performed)
	}
	key(tcell.KeyDown, 0)
	runeKey('f')
	runeKey('r')
	runeKey('R')
	if performed[1] != "o:/one" || len(refreshed) != 2 || refreshed[0] != "/one" || refreshed[1] != "" {
		t.Fatal(performed, refreshed)
	}
	runeKey('g')
	if !strings.Contains(s.notice, "not implemented") {
		t.Fatal(s.notice)
	}
	runeKey('c')
	key(tcell.KeyEscape, 0)
	if s.chord != "" {
		t.Fatal(s.chord)
	}
	model := s.model
	model.Rows = append([]table.Row{{ID: "inserted"}}, model.Rows...)
	s.replace(model)
	if s.selected != 2 {
		t.Fatal("selection moved", s.selected)
	}
	if !s.key(tcell.NewEventKey(tcell.KeyRune, 'Q', tcell.ModNone)) {
		t.Fatal("quit")
	}
}
func TestTUISimulationFormatting(t *testing.T) {
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(56, 12)
	s := tuiState{model: tuiFixture()}
	s.draw(screen)
	line := func(y int) string {
		var b strings.Builder
		for x := 0; x < 56; x++ {
			r, _, _, _ := screen.GetContent(x, y)
			b.WriteRune(r)
		}
		return b.String()
	}
	if !strings.Contains(line(0), "TalkingDuck") || !strings.Contains(line(2), "DEV-1") || !strings.Contains(line(3), "DEV-2") || !strings.Contains(line(11), "refresh") {
		t.Fatal(line(0), line(2), line(3), line(11))
	}
	_, _, selected, _ := screen.GetContent(1, 2)
	_, bg, _ := selected.Decompose()
	if bg != tcell.GetColor("#254862") {
		t.Fatal("selection missing", bg)
	}
	s.selected = 1
	s.draw(screen)
	_, _, history, _ := screen.GetContent(17, 4)
	fg, _, _ := history.Decompose()
	if fg != tuiColor("muted") {
		t.Fatal("history not dimmed", fg, line(4))
	}
	screen.SetSize(12, 4)
	s.draw(screen) // Resize must remain safe and quit stays available.
}

func TestLongChordSurvivesRefresh(t *testing.T) {
	model := tuiFixture()
	model.Rows[0].Cells[0].Lines[0].Items[0].Chord = "ff"
	s := tuiState{model: model}
	calls := 0
	s.perform = func(string, table.Item) { calls++ }
	for _, r := range "cf" {
		s.key(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	s.replace(model)
	s.key(tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModNone))
	if calls != 1 {
		t.Fatal(calls, s.chord)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, pending := range []bool{false, true} {
		s := tuiState{model: tuiFixture()}
		if pending {
			s.chord = "c"
			s.choices = s.model.Rows[0].Matches("c", "t")
		}
		if s.key(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)) {
			t.Fatal("lowercase q quit")
		}
		if !s.key(tcell.NewEventKey(tcell.KeyRune, 'Q', tcell.ModShift)) {
			t.Fatal("Shift-Q did not quit")
		}
		if !s.key(tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)) {
			t.Fatal("Ctrl-C did not quit")
		}
	}
}

func TestDirectOpenKeys(t *testing.T) {
	s := tuiState{model: tuiFixture()}
	calls := 0
	s.perform = func(verb string, item table.Item) {
		if verb != "o" {
			t.Fatal(verb)
		}
		calls++
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'o', tcell.ModNone))
	if s.chord != "" || calls != 0 {
		t.Fatal("o still starts a chord")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModNone))
	if calls != 1 {
		t.Fatal("f did not open folder")
	}
	s.key(tcell.NewEventKey(tcell.KeyRune, 't', tcell.ModNone))
	if len(s.choices) != 2 || s.verb != "o" {
		t.Fatal("t did not open ticket chooser")
	}
}
