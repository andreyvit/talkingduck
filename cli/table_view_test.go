package cli

import (
	"strings"
	"testing"
)

func TestDesktopTablePreservesTaskRowsAndTickets(t *testing.T) {
	data := map[string]any{"repositories": []any{map[string]any{
		"path": "/example/back2", "git": map[string]any{"cleanliness": "CLEAN", "tracked": 0},
		"specs": []any{map[string]any{"tickets": []string{"DEV-1", "DEV-2"}, "display_title": "<script>example</script>", "workflow": map[string]any{"stage": "HL"}}},
	}}}
	table, err := BuildTable(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 || len(table.Rows[0]) != 15 {
		t.Fatalf("wrong table shape: %v", table.Rows)
	}
	row := table.Rows[0]
	if row[0] != "back2" || row[2] != "DEV-1\nDEV-2" || row[14] != "<script>example</script>" {
		t.Fatalf("lost source cells: %v", row)
	}
	if row[1] != "?" || strings.TrimSpace(row[3]) != "?" {
		t.Fatalf("unknowns changed: %v", row)
	}
}
