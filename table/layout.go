package table

import (
	"github.com/rivo/uniseg"
	"strings"
)

type Span struct {
	X           int
	Text, Color string
	Bold        bool
}
type VisualLine struct {
	Row   int
	Spans []Span
}
type Layout struct {
	Widths []int
	Lines  []VisualLine
}

func Width(s string) int { return uniseg.StringWidth(s) }

// Format preserves semantic spans while wrapping flexible columns. All cells are
// top aligned. Fixed columns may overflow a narrow viewport; renderers clip/scroll.
func Format(t Table, width int, prefixes bool) Layout {
	result := Layout{Widths: make([]int, len(t.Columns))}
	for j, c := range t.Columns {
		result.Widths[j] = c.MinWidth
		if prefixes {
			result.Widths[j] += Width(c.Prefix)
		}
		if c.Flex {
			continue
		}
		for _, r := range t.Rows {
			if j >= len(r.Cells) {
				continue
			}
			for _, l := range r.Cells[j].Lines {
				n := Width(l.Text())
				if prefixes && l.Text() != "" {
					n += Width(c.Prefix)
				}
				result.Widths[j] = max(result.Widths[j], n)
			}
		}
	}
	remaining := width - max(0, len(t.Columns)-1)
	flex := 0
	for j, c := range t.Columns {
		if c.Flex {
			flex++
		} else {
			remaining -= result.Widths[j]
		}
	}
	for j, c := range t.Columns {
		if c.Flex {
			result.Widths[j] = max(c.MinWidth, remaining/max(1, flex))
		}
	}
	for ri, row := range t.Rows {
		cells := make([][][]Span, len(t.Columns))
		height := 1
		for ci, col := range t.Columns {
			if ci >= len(row.Cells) {
				continue
			}
			for _, line := range row.Cells[ci].Lines {
				spans := []Span{}
				x := 0
				if prefixes && col.Prefix != "" && line.Text() != "" {
					color := "dim"
					if row.State == "pushed" {
						color = "muted"
					}
					spans = append(spans, Span{Text: col.Prefix, Color: color})
					x = Width(col.Prefix)
				}
				for _, item := range line.Items {
					if item.Hidden {
						continue
					}
					graphemes := uniseg.NewGraphemes(item.Text)
					for graphemes.Next() {
						text := graphemes.Str()
						w := Width(text)
						if col.Flex && x+w > result.Widths[ci] && x > 0 {
							cells[ci] = append(cells[ci], spans)
							spans = nil
							x = 0
						}
						color := item.Color
						if row.State == "pushed" {
							color = "muted"
						}
						if n := len(spans); n > 0 && spans[n-1].Color == color && spans[n-1].Bold == item.Bold && spans[n-1].X+Width(spans[n-1].Text) == x {
							spans[n-1].Text += text
						} else {
							spans = append(spans, Span{X: x, Text: text, Color: color, Bold: item.Bold})
						}
						x += w
					}
				}
				offset := 0
				if col.Align == "right" {
					offset = max(0, result.Widths[ci]-x)
				} else if col.Align == "center" {
					offset = max(0, (result.Widths[ci]-x)/2)
				}
				for k := range spans {
					if !(prefixes && col.Prefix != "" && k == 0) {
						spans[k].X += offset
					}
				}
				cells[ci] = append(cells[ci], spans)
			}
			height = max(height, len(cells[ci]))
		}
		for line := 0; line < height; line++ {
			visual := VisualLine{Row: ri}
			x := 0
			for ci := range t.Columns {
				if line < len(cells[ci]) {
					for _, s := range cells[ci][line] {
						s.X += x
						visual.Spans = append(visual.Spans, s)
					}
				}
				x += result.Widths[ci] + 1
			}
			result.Lines = append(result.Lines, visual)
		}
	}
	return result
}
func Plain(l Layout) string {
	lines := []string{}
	for _, line := range l.Lines {
		var b strings.Builder
		x := 0
		for _, s := range line.Spans {
			if s.X > x {
				b.WriteString(strings.Repeat(" ", s.X-x))
			}
			b.WriteString(s.Text)
			x = s.X + Width(s.Text)
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return strings.Join(lines, "\n")
}
