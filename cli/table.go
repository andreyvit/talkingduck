package cli

import (
	"fmt"
	"golang.org/x/term"
	"golang.org/x/text/width"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

func cellWidth(s string) int {
	n := 0
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || r == '\ufe0e' || r == '\ufe0f' {
			continue
		}
		kind := width.LookupRune(r).Kind()
		if kind == width.EastAsianWide || kind == width.EastAsianFullwidth {
			n += 2
		} else {
			n++
		}
	}
	return n
}
func clean(s string) string { return strings.Join(strings.Fields(s), " ") }
func wrap(text string, size int) []string {
	result := []string{}
	var current strings.Builder
	n := 0
	for _, r := range clean(text) {
		w := cellWidth(string(r))
		if n > 0 && n+w > size {
			result = append(result, current.String())
			current.Reset()
			n = 0
		}
		current.WriteRune(r)
		n += w
	}
	return append(result, current.String())
}
func display(v any) string {
	if v == nil {
		return "?"
	}
	return fmt.Sprint(v)
}

// Numeric cells contain intentional padding; text cells are whitespace-normalized.
func tableCell(s string, column int) string {
	if column == 3 || column == 6 || column == 9 || (column >= 10 && column <= 14) {
		return s
	}
	return clean(s)
}

func numeric(v any) string { return fmt.Sprintf("%2s", display(v)) }

func uberMarker(enabled bool) string {
	if enabled {
		return "u"
	}
	return ""
}

func tableRows(report tableReport) [][]string {
	rows := [][]string{}
	for _, repo := range report.Repositories {
		if repo.HideSummary && (repo.Git.Error != "" || len(repo.Specs) == 0) {
			continue
		}
		name := filepath.Base(repo.Path)
		g := repo.Git
		if repo.Cache != nil && repo.Cache.ObservedAt.IsZero() && repo.Cache.Error == "" && (repo.Cache.State == "pending" || repo.Cache.State == "updating") {
			row := make([]string, 16)
			row[0], row[4] = name, "SCANNING"
			rows = append(rows, row)
			continue
		}
		if g.Error != "" {
			rows = append(rows, []string{name, "--", "--", " 0", "--", "", "--", "?", "?", " ?/ ?", "ERROR", "", "", "⇣ ?", "⇡ ?", g.Error})
			continue
		}
		tracked := "±" + numeric(g.Tracked)
		untracked := "?" + numeric(g.Untracked)
		info := "ℹ︎" + numeric(g.Informational)
		behind, ahead := "⇣"+numeric(g.Behind), "⇡"+numeric(g.Ahead)
		specs := repo.Specs
		if needsCurrentSummary(repo) {
			rows = append(rows, []string{name, "--", "--", " 0", g.Cleanliness, "", " 0/ 0", "", "", " 0/ 0", tracked, untracked, info, behind, ahead, worktreeSummary(repo)})
			name = ""
		}
		for _, s := range specs {
			w, p := s.Workflow, s.PrimarySession
			agent := map[string]string{"codex": "Codex", "claude": "Claude", "grok": "Grok"}[p.Engine]
			if agent == "" {
				agent = "?"
			}
			if s.PrimaryConflict {
				agent += "!"
			}
			ids := []string{}
			for _, id := range s.Tickets {
				ids = append(ids, id)
			}
			ticket := strings.Join(ids, "\n")
			if ticket == "" {
				ticket = "--"
			}
			rows = append(rows, []string{name, agent, ticket, numeric(w.Round), w.Stage, uberMarker(s.UberReviews), numeric(w.SubtasksDone) + "/" + numeric(w.SubtasksTotal), w.Progress, w.Interaction, numeric(w.Questions) + "/" + numeric(w.Suggestions), tracked, untracked, info, behind, ahead, s.DisplayTitle})
			name = "" // Show the worktree once; AI output restores its full path.
		}
	}
	return rows
}

func render(report tableReport, size int, colors bool) string {
	rows := tableRows(report)
	widths := []int{6, 6, 8, 2, 4, 1, 5, 1, 1, 5, 3, 3, 3, 3, 3}
	for _, row := range rows {
		for i, value := range row[:len(widths)] {
			for _, line := range strings.Split(value, "\n") {
				widths[i] = max(widths[i], cellWidth(tableCell(line, i)))
			}
		}
	}
	total := len(widths)
	for _, w := range widths {
		total += w
	}
	lastWidth := max(8, size-total)
	out := []string{}
	for _, row := range rows {
		cells := [][]string{}
		height := 0
		for i, value := range row {
			var lines []string
			if i == len(widths) {
				lines = wrap(value, lastWidth)
			} else {
				for _, line := range strings.Split(value, "\n") {
					lines = append(lines, tableCell(line, i))
				}
			}
			cells = append(cells, lines)
			height = max(height, len(lines))
		}
		for line := 0; line < height; line++ {
			values := []string{}
			for i, cell := range cells {
				value := ""
				if line < len(cell) {
					value = cell[line]
				}
				if i < len(widths) {
					value += strings.Repeat(" ", max(0, widths[i]-cellWidth(value)))
				}
				if colors {
					if row[4] == "PUSHED" {
						value = paint(value, 245)
					} else {
						value = colorCell(value, i)
					}
				}
				values = append(values, value)
			}
			out = append(out, strings.TrimRight(strings.Join(values, " "), " "))
		}
	}
	return strings.Join(out, "\n")
}
func tableWidth(requested int, out io.Writer) int {
	if requested > 0 {
		return requested
	}
	if n, e := strconv.Atoi(os.Getenv("COLUMNS")); e == nil && n > 0 {
		return n
	}
	if f, ok := out.(*os.File); ok {
		if w, _, e := term.GetSize(int(f.Fd())); e == nil && w > 0 {
			return w
		}
	}
	return 120
}
