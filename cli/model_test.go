package cli

import "testing"

func TestModelActionsAndStates(t *testing.T) {
	data := map[string]any{"repositories": []any{map[string]any{"path": "/example/back2", "specs": []any{
		map[string]any{"path": ".spec/a.md", "display_title": "Example", "tickets": []string{"DEV-1", "DEV-2"}, "workflow": map[string]any{"stage": "EX", "progress": "▶"}},
		map[string]any{"path": ".spec/b.md", "display_title": "Old", "workflow": map[string]any{"stage": "PUSHED"}},
	}}}}
	view, err := BuildTable(data)
	if err != nil {
		t.Fatal(err)
	}
	rows := view.Model.Rows
	if rows[0].State != "running" || rows[1].State != "pushed" {
		t.Fatal(rows)
	}
	tickets := rows[0].Matches("c", "t")
	if len(tickets) != 2 || *tickets[1].Copy != "DEV-2" {
		t.Fatal(tickets)
	}
	folders := rows[1].Matches("o", "f")
	if len(folders) != 1 || !folders[0].Hidden || folders[0].Link.Target != "/example/back2" {
		t.Fatal(folders)
	}
	if rows[0].Matches("o", "s")[0].Link.Target != "/example/back2/.spec/a.md" {
		t.Fatal(rows[0])
	}
}
