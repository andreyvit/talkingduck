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

func TestNewAgentActionsWithoutSessions(t *testing.T) {
	folder := "/example/repo/.worktrees/child"
	for _, specs := range [][]any{nil, {map[string]any{"path": ".spec/task.md"}}} {
		view, err := BuildTable(map[string]any{"repositories": []any{map[string]any{"path": folder, "specs": specs}}})
		if err != nil {
			t.Fatal(err)
		}
		if len(view.Model.Rows) != 1 {
			t.Fatal(view.Model.Rows)
		}
		row := view.Model.Rows[0]
		for chord, kind := range map[string]string{"Ax": "new-codex", "Ac": "new-claude", "Ag": "new-grok"} {
			items := row.Matches("o", chord)
			if len(items) != 1 || items[0].Link.Kind != kind || items[0].Link.Target != folder {
				t.Fatalf("%s: %+v", chord, items)
			}
		}
	}
}

func TestPickAgentReusesTaskSessionBeforeCreating(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"
	view, err := BuildTable(map[string]any{"repositories": []any{map[string]any{
		"path": "/example/repo", "specs": []any{map[string]any{"path": ".spec/task.md",
			"primary_session": map[string]any{"engine": "codex", "client": "desktop", "id": id},
		}},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	row := view.Model.Rows[0]
	x := row.Matches("o", "Ax")
	c := row.Matches("o", "Ac")
	if len(x) != 1 || x[0].Link.Kind != "agent" || x[0].Link.Target != "codex://threads/"+id {
		t.Fatal(x)
	}
	if len(c) != 1 || c[0].Link.Kind != "new-claude" || c[0].Link.Target != "/example/repo" {
		t.Fatal(c)
	}
}
