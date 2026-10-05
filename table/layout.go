package table

import (
	"github.com/rivo/uniseg"
	"strings"
)

type Span struct {
	X           int
	Text, Color string
	Bold        bool
	Source      *ItemRef
	CopyControl bool
}
type VisualLine struct {
	Row   int
	Spans []Span
	Hover []HoverRegion
}
type Layout struct {
	Widths []int
	Lines  []VisualLine
}

func Width(s string) int { return uniseg.StringWidth(s) }

// Format preserves semantic spans while wrapping flexible columns. All cells are
// top aligned. Fixed columns may overflow a narrow viewport; renderers clip/scroll.
func Format(t Table, width int, prefixes bool) Layout {
	return FormatWithOptions(t, width, FormatOptions{Prefixes: prefixes})
}

// FormatOptions adds presentation controls without changing the underlying data.
type FormatOptions struct {
	Prefixes    bool
	CopyControl string
}

func FormatWithOptions(t Table, width int, options FormatOptions) Layout {
	prefixes := options.Prefixes
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
				for _, item := range l.Items {
					if !item.Hidden && item.Copy != nil && options.CopyControl != "" {
						n += Width(options.CopyControl)
					}
				}
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
			for li, line := range row.Cells[ci].Lines {
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
				for ii, item := range line.Items {
					if item.Hidden {
						continue
					}
					ref := &ItemRef{Row: ri, Column: ci, Line: li, Item: ii}
					parts := []string{item.Text}
					if item.Copy != nil && options.CopyControl != "" {
						parts = append(parts, options.CopyControl)
					}
					for part, content := range parts {
						graphemes := uniseg.NewGraphemes(content)
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
							if n := len(spans); n > 0 && spans[n-1].Source == ref && spans[n-1].CopyControl == (part == 1) && spans[n-1].Color == color && spans[n-1].Bold == item.Bold && spans[n-1].X+Width(spans[n-1].Text) == x {
								spans[n-1].Text += text
							} else {
								spans = append(spans, Span{X: x, Text: text, Color: color, Bold: item.Bold, Source: ref, CopyControl: part == 1})
							}
							x += w
						}
					}
				}
				offset := AlignmentOffset(col.Align, result.Widths[ci], x)
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
					spans := cells[ci][line]
					for _, span := range spans {
						if span.Source == nil {
							continue
						}
						source := span.Source
						count := 0
						for _, item := range row.Cells[ci].Lines[source.Line].Items {
							if !item.Hidden {
								count++
							}
						}
						if count == 1 {
							visual.Hover = append(visual.Hover, HoverRegion{X: x, Width: result.Widths[ci], Source: source})
						}
						break
					}
					for _, s := range spans {
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

// AlignmentOffset positions content within a column in terminal cells.
func AlignmentOffset(align string, width, contentWidth int) int {
	switch align {
	case "right":
		return max(0, width-contentWidth)
	case "center":
		return max(0, (width-contentWidth)/2)
	default:
		return 0
	}
}
