package table

// ItemRef identifies an item within a layout's source table.
type ItemRef struct {
	Row, Column, Line, Item int
}

func (r ItemRef) Resolve(t Table) Item {
	return t.Rows[r.Row].Cells[r.Column].Lines[r.Line].Items[r.Item]
}

type Hit struct {
	Row         int
	Source      *ItemRef
	CopyControl bool
}

// HitTest uses terminal cell coordinates, including both cells of wide glyphs.
// Blank space within a row selects that row but has no item action.
func (l Layout) HitTest(x, y int) (Hit, bool) {
	if x < 0 || y < 0 || y >= len(l.Lines) {
		return Hit{}, false
	}
	line := l.Lines[y]
	hit := Hit{Row: line.Row}
	for _, span := range line.Spans {
		if x >= span.X && x < span.X+Width(span.Text) {
			hit.Source, hit.CopyControl = span.Source, span.CopyControl
			break
		}
	}
	return hit, true
}

// HoverRegion extends a single item's hover across its cell line, without
// turning blank space into a link or copy button.
type HoverRegion struct {
	X, Width int
	Source   *ItemRef
}

func (l Layout) HoverTest(x, y int) (Hit, bool) {
	hit, ok := l.HitTest(x, y)
	if !ok || hit.Source != nil {
		return hit, ok
	}
	for _, region := range l.Lines[y].Hover {
		if x >= region.X && x < region.X+region.Width {
			hit.Source = region.Source
			break
		}
	}
	return hit, ok
}
