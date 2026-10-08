package cli

import (
	"path/filepath"
	"regexp"
	"strings"
	"talkingduck/table"
)

var ticketID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

func pointer(s string) *string { return &s }
func modelTable(view TableView) table.Table {
	columns := []table.Column{
		{ID: "folder", Label: "Folder", MinWidth: 6}, {ID: "agent", Label: "Agent", MinWidth: 6}, {ID: "ticket", Label: "Ticket", MinWidth: 8},
		{ID: "revision", Label: "Rev", Align: "right", MinWidth: 2}, {ID: "stage", Label: "Stage", MinWidth: 4}, {ID: "uber", Label: "", MinWidth: 1},
		{ID: "tasks", Label: "Tasks", Align: "right", MinWidth: 5}, {ID: "progress", Label: "", MinWidth: 1}, {ID: "input", Label: "", MinWidth: 1},
		{ID: "questions", Label: "Q", Align: "right", MinWidth: 2}, {ID: "tracked", Label: "Mod", Align: "right", MinWidth: 2, Prefix: "±"},
		{ID: "untracked", Label: "New", Align: "right", MinWidth: 2, Prefix: "?"}, {ID: "markdown", Label: ".md", Align: "right", MinWidth: 2, Prefix: "ℹ︎"},
		{ID: "behind", Label: "↓", Align: "right", MinWidth: 2, Prefix: "⇣"}, {ID: "ahead", Label: "↑", Align: "right", MinWidth: 2, Prefix: "⇡"}, {ID: "title", Label: "Title", Flex: true, MinWidth: 8},
	}
	// Accept the earlier projection while the Uber/Q change is developed separately.
	if len(view.Rows) > 0 && len(view.Rows[0]) == 15 {
		columns = append(columns[:5], columns[6:]...)
	}
	// Source rows retain their legacy order; both UIs use this presentation order.
	order := []string{"folder", "agent", "ticket", "stage", "uber", "progress", "input", "revision", "tasks", "questions", "tracked", "markdown", "behind", "ahead", "title"}
	indices := make([]int, 0, len(columns))
	model := table.Table{Rows: []table.Row{}}
	for _, id := range order {
		for i, col := range columns {
			if col.ID == id {
				indices = append(indices, i)
				model.Columns = append(model.Columns, col)
				break
			}
		}
	}
	for n, raw := range view.Rows {
		row := table.Row{Group: view.WorktreePaths[n], ID: view.WorktreePaths[n] + ":" + view.SpecPaths[n]}
		row.Actions = []table.Item{
			{Text: "Editor", Chord: "e", Link: &table.Action{Kind: "editor", Target: row.Group}},
			{Text: "Git", Chord: "g", Link: &table.Action{Kind: "git", Target: row.Group}},
		}
		if view.SpecPaths[n] == "" {
			row.Archive = &table.Action{Kind: "archive", Target: ""}
		} else if relative, err := filepath.Rel(row.Group, view.SpecPaths[n]); err == nil {
			row.Archive = &table.Action{Kind: "archive", Target: filepath.ToSlash(relative)}
		}
		for _, j := range indices {
			value := raw[j]
			col := columns[j]
			if col.ID == "tracked" {
				value = modifiedCount(value, raw[j+1])
			}
			cell := table.Cell{}
			for _, text := range strings.Split(value, "\n") {
				text = strings.TrimSpace(text)
				if col.Prefix != "" {
					text = strings.TrimSpace(strings.TrimPrefix(text, col.Prefix))
				}
				item := table.Item{Text: text, Color: itemColor(col.ID, text)}
				switch col.ID {
				case "questions":
					if text == "0" {
						item.Text = "-"
					}
				case "folder":
					item.Text = filepath.Base(row.Group)
					item.Hidden = text == ""
					item.Bold = true
					item.Chord = "f"
					item.Copy = pointer(row.Group)
					item.Link = &table.Action{Kind: "folder", Target: row.Group}
					item.Tooltip = "Open in Finder: " + row.Group
				case "agent":
					item.Chord = "a"
					if view.AgentLinks[n] != "" {
						item.Link = &table.Action{Kind: "agent", Target: view.AgentLinks[n]}
						item.Tooltip = "Open this chat"
					}
				case "ticket":
					if ticketID.MatchString(text) {
						item.Chord = "t"
						item.Copy = pointer(text)
						item.Link = &table.Action{Kind: "ticket", Target: text}
						item.Tooltip = "Open ticket"
					}
				case "title":
					if view.SpecPaths[n] != "" {
						item.Chord = "s"
						item.Copy = pointer(text)
						item.Link = &table.Action{Kind: "spec", Target: view.SpecPaths[n]}
						item.Tooltip = "Open spec"
					}
				}
				cell.Lines = append(cell.Lines, table.Line{Items: []table.Item{item}})
			}
			row.Cells = append(row.Cells, cell)
		}
		get := func(id string) string {
			for i, c := range model.Columns {
				if c.ID == id {
					return row.Cells[i].Text()
				}
			}
			return ""
		}
		switch {
		case get("stage") == "PUSHED":
			row.State = "pushed"
		case get("input") == "!":
			row.State = "attention"
		case get("progress") == "▶":
			row.State = "running"
		case get("progress") == "✓":
			row.State = "attention"
		case get("tracked") == "ERROR":
			row.State = "error"
		}
		model.Rows = append(model.Rows, row)
	}
	return model
}
func itemColor(id, text string) string {
	if text == "?" || text == "--" {
		return "dim"
	}
	switch id {
	case "agent":
		return map[string]string{"Codex": "cyan", "Claude": "orange", "Grok": "purple"}[strings.TrimSuffix(text, "!")]
	case "stage":
		return map[string]string{"INV": "blue", "HL": "blue", "LL": "purple", "EX": "cyan", "FIN": "green", "SQ": "purple", "PUSH": "yellow", "PUSHED": "muted", "REBA": "orange"}[text]
	case "progress", "input":
		return map[string]string{"▶": "cyan", "✓": "green", "!": "yellow"}[text]
	case "revision", "questions", "tracked", "untracked", "markdown", "behind", "ahead":
		if text == "0" {
			return "dim"
		}
		if text == "ERROR" {
			return "red"
		}
		return map[string]string{"revision": "purple", "questions": "yellow", "tracked": "yellow", "untracked": "green", "markdown": "blue", "behind": "red", "ahead": "cyan"}[id]
	}
	return ""
}
