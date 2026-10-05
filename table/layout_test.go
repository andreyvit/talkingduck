package table

import "testing"

func text(s string) Cell { return Cell{Lines: []Line{{Items: []Item{{Text: s}}}}} }
func TestLayoutAlignmentAndMultiline(t *testing.T) {
	model := Table{Columns: []Column{{ID: "name", MinWidth: 4}, {ID: "n", Align: "right", MinWidth: 3}, {ID: "title", Flex: true, MinWidth: 4}}, Rows: []Row{{Cells: []Cell{text("界"), {Lines: []Line{{Items: []Item{{Text: "2"}}}, {Items: []Item{{Text: "13"}}}}}, text("abcdefgh")}}}}
	got := Plain(Format(model, 15, false))
	want := "界     2 abcdef\n      13 gh"
	if got != want {
		t.Fatalf("%q != %q", got, want)
	}
}
func TestItemsActionsAndMutedHistory(t *testing.T) {
	copy := "DEV-12"
	row := Row{State: "pushed", Cells: []Cell{{Lines: []Line{{Items: []Item{{Text: "hidden", Hidden: true, Chord: "f", Copy: &copy}, {Text: "DEV-12", Color: "cyan", Chord: "t", Copy: &copy, Link: &Action{Kind: "ticket", Target: copy}}, {Text: " note"}}}}}}}
	if len(row.Matches("c", "f")) != 1 || len(row.Matches("o", "f")) != 0 || *row.Matches("c", "t")[0].Copy != "DEV-12" {
		t.Fatal(row)
	}
	layout := Format(Table{Columns: []Column{{MinWidth: 20}}, Rows: []Row{row}}, 30, false)
	if Plain(layout) != "DEV-12 note" {
		t.Fatal(Plain(layout))
	}
	for _, span := range layout.Lines[0].Spans {
		if span.Color != "muted" {
			t.Fatal(span)
		}
	}
}
