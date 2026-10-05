package cli

import (
	"path/filepath"
	"strings"
	"talkingduck/inventory"
	"testing"
)

func TestArchivedSummaryIsHiddenAcrossOutputs(t *testing.T) {
	folder := t.TempDir()
	cache, err := inventory.New(inventory.Options{Paths: []string{folder}, HistoryPath: filepath.Join(t.TempDir(), "state.json")})
	if err != nil {
		t.Fatal(err)
	}
	if err = cache.ArchiveRow(folder, ""); err != nil {
		t.Fatal(err)
	}
	view, err := BuildTable(cache.Snapshot().Overview)
	if err != nil || len(view.Model.Rows) != 0 || len(view.WorktreePaths) != 0 {
		t.Fatal(view, err)
	}
	ai, err := FormatAI(cache.Snapshot().Overview)
	if err != nil || strings.Contains(ai, folder) {
		t.Fatal(ai, err)
	}
}
