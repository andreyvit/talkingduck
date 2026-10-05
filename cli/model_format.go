package cli

import (
	"strings"
	"talkingduck/table"
)

var semanticANSI = map[string]int{"dim": 242, "muted": 245, "cyan": 81, "orange": 209, "purple": 141, "green": 114, "yellow": 221, "blue": 75, "red": 203}

func formatModel(model table.Table, width int, colors bool) string {
	layout := table.Format(model, width, true)
	if !colors {
		return table.Plain(layout)
	}
	lines := []string{}
	for _, line := range layout.Lines {
		var b strings.Builder
		x := 0
		for _, s := range line.Spans {
			if s.X > x {
				b.WriteString(strings.Repeat(" ", s.X-x))
			}
			color := 15
			if c, ok := semanticANSI[s.Color]; ok {
				color = c
			}
			b.WriteString(paint(s.Text, color))
			x = s.X + table.Width(s.Text)
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return strings.Join(lines, "\n")
}
