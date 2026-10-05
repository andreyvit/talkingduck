// Package table describes and lays out tables without depending on a UI toolkit.
package table

import "strings"

type Column struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Align    string `json:"align"` // left, right, center
	MinWidth int    `json:"minWidth"`
	Flex     bool   `json:"flex,omitempty"`
	Prefix   string `json:"prefix,omitempty"` // Optional compact-terminal counter symbol.
}
type Action struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
}
type Item struct {
	Text    string  `json:"text"`
	Color   string  `json:"color,omitempty"`
	Bold    bool    `json:"bold,omitempty"`
	Hidden  bool    `json:"hidden,omitempty"` // Grouped label; actions remain available.
	Chord   string  `json:"chord,omitempty"`
	Tooltip string  `json:"tooltip,omitempty"`
	Link    *Action `json:"link,omitempty"`
	Copy    *string `json:"copy,omitempty"`
}
type Line struct {
	Items []Item `json:"items"`
}
type Cell struct {
	Lines []Line `json:"lines"`
}
type Row struct {
	ID    string `json:"id"`
	Group string `json:"group"`           // Full worktree path; refresh target, not display text.
	State string `json:"state,omitempty"` // running, attention, pushed, error
	Cells []Cell `json:"cells"`
}
type Table struct {
	Columns []Column `json:"columns"`
	Rows    []Row    `json:"rows"`
}

func (l Line) Text() string {
	var b strings.Builder
	for _, i := range l.Items {
		if !i.Hidden {
			b.WriteString(i.Text)
		}
	}
	return b.String()
}
func (c Cell) Text() string {
	lines := []string{}
	for _, l := range c.Lines {
		lines = append(lines, l.Text())
	}
	return strings.Join(lines, "\n")
}

// Matches returns all actionable items for a chord, including hidden group labels.
func (r Row) Matches(verb, key string) []Item {
	var result []Item
	for _, c := range r.Cells {
		for _, l := range c.Lines {
			for _, i := range l.Items {
				if i.Chord == key && ((verb == "o" && i.Link != nil) || (verb == "c" && i.Copy != nil)) {
					result = append(result, i)
				}
			}
		}
	}
	return result
}
func (r Row) Chords(verb string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, c := range r.Cells {
		for _, l := range c.Lines {
			for _, i := range l.Items {
				if i.Chord != "" && !seen[i.Chord] && len(r.Matches(verb, i.Chord)) > 0 {
					seen[i.Chord] = true
					keys = append(keys, i.Chord)
				}
			}
		}
	}
	return keys
}
