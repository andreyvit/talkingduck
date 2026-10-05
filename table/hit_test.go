package table

import "testing"

func TestInteractiveLayoutTargets(t *testing.T) {
	copyText := "ticket"
	model := Table{Columns: []Column{{MinWidth: 8, Align: "right"}, {MinWidth: 3, Flex: true}}, Rows: []Row{{Cells: []Cell{
		{Lines: []Line{{Items: []Item{{Text: "界", Copy: &copyText}, {Text: "x"}}}}},
		{Lines: []Line{{Items: []Item{{Text: "abcdef", Copy: &copyText}, {Text: "hidden", Hidden: true, Copy: &copyText}}}}},
	}}}}
	layout := FormatWithOptions(model, 12, FormatOptions{CopyControl: "©"})
	// Right aligned: four spaces, wide glyph, copy control, adjacent item.
	for _, x := range []int{4, 5} {
		hit, ok := layout.HitTest(x, 0)
		if !ok || hit.Source == nil || hit.Source.Item != 0 || hit.CopyControl {
			t.Fatalf("wide glyph at %d: %+v", x, hit)
		}
	}
	hit, _ := layout.HitTest(6, 0)
	if !hit.CopyControl || hit.Source.Resolve(model).Text != "界" {
		t.Fatal(hit)
	}
	hit, _ = layout.HitTest(7, 0)
	if hit.Source.Item != 1 || hit.CopyControl {
		t.Fatal("adjacent item merged", hit)
	}
	hit, _ = layout.HitTest(9, 2)
	if !hit.CopyControl || hit.Source.Column != 1 {
		t.Fatal("wrapped copy control", hit, Plain(layout))
	}
	if len(layout.Lines) != 3 {
		t.Fatal("hidden item allocated space", Plain(layout))
	}
	if _, ok := layout.HitTest(0, 3); ok {
		t.Fatal("outside hit")
	}
}

func TestSingleItemWhitespaceHover(t *testing.T) {
	model := Table{Columns: []Column{{MinWidth: 8}, {MinWidth: 8}}, Rows: []Row{{Cells: []Cell{
		{Lines: []Line{{Items: []Item{{Text: "one"}}}}},
		{Lines: []Line{{Items: []Item{{Text: "a"}, {Text: "b"}}}}},
	}}}}
	layout := Format(model, 17, false)
	hit, _ := layout.HoverTest(7, 0)
	if hit.Source == nil || hit.Source.Resolve(model).Text != "one" {
		t.Fatal("single-item whitespace did not hover", hit)
	}
	hit, _ = layout.HitTest(7, 0)
	if hit.Source != nil {
		t.Fatal("hover whitespace became clickable")
	}
	hit, _ = layout.HoverTest(16, 0)
	if hit.Source != nil {
		t.Fatal("multi-item whitespace chose an item")
	}
}
