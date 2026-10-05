//go:build darwin && cgo

package main

import (
	"path/filepath"
	"talkingduck/inventory"
	"testing"
)

func TestArchivePublishesWithoutScan(t *testing.T) {
	folder := t.TempDir()
	cache, err := inventory.New(inventory.Options{Paths: []string{folder}, HistoryPath: filepath.Join(t.TempDir(), "state.json")})
	if err != nil {
		t.Fatal(err)
	}
	var published inventoryState
	v := &inventoryView{cache: cache, emit: func(state inventoryState) { published = state }}
	if err = v.archiveRow(folder, ""); err != nil {
		t.Fatal(err)
	}
	if len(published.Model.Rows) != 0 {
		t.Fatal("archived summary still published", published.Model)
	}
	if cache.Snapshot().Details.Repositories[0].Cache.State != "pending" {
		t.Fatal("archive ran a scan")
	}
}
