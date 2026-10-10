package cli

import "testing"

func TestLinkedWorktreePresentation(t *testing.T) {
	data := map[string]any{"repositories": []any{
		map[string]any{"path": "/repo"},
		map[string]any{"path": "/other"},
		map[string]any{"path": "/elsewhere/long-linked-checkout-name", "parent_path": "/repo"},
	}}
	view, err := BuildTable(data)
	if err != nil {
		t.Fatal(err)
	}
	rows := view.Model.Rows
	if len(rows) != 3 || rows[1].Group != "/elsewhere/long-linked-checkout-name" || rows[2].Group != "/other" {
		t.Fatalf("order: %+v", rows)
	}
	for i, col := range view.Model.Columns {
		if col.ID == "folder" {
			c := rows[1].Cells[i]
			if c.Indent != 2 || c.WrapWidth != 14 || !c.Small {
				t.Fatalf("style: %+v", c)
			}
			if rows[0].Cells[i].Small {
				t.Fatal("parent got child styling")
			}
		}
	}
	item := rows[1].Matches("o", "f")[0]
	if item.Link.Target != rows[1].Group || *item.Copy != rows[1].Group {
		t.Fatal("parent stole child action")
	}
}
