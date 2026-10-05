package cli

import (
	"path/filepath"
	"talkingduck/inventory"
	"testing"
)

func TestInitialScanStage(t *testing.T) {
	cache, err := inventory.New(inventory.Options{Paths: []string{filepath.Join(t.TempDir(), "missing")}, NoAgents: true})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := cache.Snapshot()
	if snapshot.Failed() {
		t.Fatal("pending scan treated as failure")
	}
	view, err := BuildTable(snapshot.Overview)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Model.Rows) != 1 {
		t.Fatal(view)
	}
	for n, col := range view.Model.Columns {
		text := view.Model.Rows[0].Cells[n].Text()
		if col.ID == "stage" {
			if text != "SCANNING" {
				t.Fatal(text)
			}
		} else if col.ID != "folder" && text != "" {
			t.Fatalf("%s = %q", col.ID, text)
		}
	}
	_ = cache.Update(cache.Paths()[0])
	view, err = BuildTable(cache.Snapshot().Overview)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, cell := range view.Model.Rows[0].Cells {
		if cell.Text() == "ERROR" {
			found = true
		}
	}
	if !found {
		t.Fatal("real scan error hidden", view.Rows)
	}
}
