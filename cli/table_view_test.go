package cli

import (
	"strings"
	"testing"
)

func TestDesktopTablePreservesTaskRowsAndTickets(t *testing.T) {
	data := map[string]any{"repositories": []any{map[string]any{
		"path": "/example/back2", "git": map[string]any{"cleanliness": "CLEAN", "tracked": 0},
		"specs": []any{map[string]any{"path": ".spec/example.md", "tickets": []string{"DEV-1", "DEV-2"}, "display_title": "<script>example</script>", "workflow": map[string]any{"stage": "HL"}}},
	}}}
	table, err := BuildTable(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 || len(table.Rows[0]) != 15 {
		t.Fatalf("wrong table shape: %v", table.Rows)
	}
	if len(table.SpecPaths) != 1 || table.SpecPaths[0] != "/example/back2/.spec/example.md" {
		t.Fatalf("wrong spec paths: %v", table.SpecPaths)
	}
	row := table.Rows[0]
	if row[0] != "back2" || row[2] != "DEV-1\nDEV-2" || row[14] != "<script>example</script>" {
		t.Fatalf("lost source cells: %v", row)
	}
	if row[1] != "?" || strings.TrimSpace(row[3]) != "?" {
		t.Fatalf("unknowns changed: %v", row)
	}
}

func TestTablePathsAlignWithEmptyAndFailedCheckouts(t *testing.T) {
	table, err := BuildTable(map[string]any{"repositories": []any{
		map[string]any{"path": "/empty", "specs": []any{}},
		map[string]any{"path": "/failed", "git": map[string]any{"error": "unavailable"}, "specs": []any{map[string]any{"path": "ignored.md"}}},
		map[string]any{"path": "/repo", "specs": []any{map[string]any{"path": ".spec/one.md"}, map[string]any{"path": ".spec/two.md"}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"", "", "/repo/.spec/one.md", "/repo/.spec/two.md"}
	if len(table.Rows) != len(want) || len(table.SpecPaths) != len(want) {
		t.Fatal(table)
	}
	for i, p := range want {
		if table.SpecPaths[i] != p {
			t.Fatalf("row %d: %q, want %q", i, table.SpecPaths[i], p)
		}
	}
}
