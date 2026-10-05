package table

import "testing"

func TestInteractiveLayoutTargets(t *testing.T) {
	copyText := "ticket"
	model := Table{Columns: []Column{{MinWidth: 8, Align: "right"}, {MinWidth: 3, Flex: true}}, Rows: []Row{{Cells: []Cell{
		{Lines: []Line{{Items: []Item{{Text: "界", Copy: &copyText}, {Text: "x"}}}}},
		{Lines: []Line{{Items: []Item{{Text: "abcdef", Copy: &copyText}, {Text: "hidden", Hidden: true, Copy: &copyText}}}}},
	}}}}
	layout := FormatWithOptions(model, 12, FormatOptions{CopyControl: "©"})
	// Right aligned: three spaces, wide glyph, copy control, adjacent item.
	for _, x := range []int{3, 4} {
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
	hit, _ = layout.HitTest(10, 2)
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
