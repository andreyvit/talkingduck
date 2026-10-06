//go:build darwin && cgo

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"talkingduck/inventory"
	"testing"
)

func TestUnarchiveRefreshesAndPublishes(t *testing.T) {
	folder := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", folder).CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	for _, args := range [][]string{
		{"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Initial"},
		{"update-ref", "refs/remotes/origin/main", "HEAD"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", folder}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out), err)
		}
	}

	if err := os.Mkdir(filepath.Join(folder, ".spec"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, ".spec/work.md"), []byte("# Work"), 0600); err != nil {
		t.Fatal(err)
	}
	cache, err := inventory.New(inventory.Options{Paths: []string{folder}, NoAgents: true, HistoryPath: filepath.Join(t.TempDir(), "state.json")})
	if err != nil {
		t.Fatal(err)
	}
	if err = cache.Update(folder); err != nil {
		t.Fatal(err)
	}
	if err = cache.ArchiveRow(folder, ".spec/work.md"); err != nil {
		t.Fatal(err)
	}
	if err = cache.Update(folder); err != nil {
		t.Fatal(err)
	}
	var published inventoryState
	changed := false
	v := &inventoryView{cache: cache, emit: func(s inventoryState) { published = s }, settingsChanged: func() { changed = true }}
	if err = v.unarchiveRow(folder, ".spec/work.md"); err != nil {
		t.Fatal(err)
	}
	if !changed || len(published.Model.Rows) != 1 || len(cache.Snapshot().Details.Repositories[0].Specs) != 1 {
		t.Fatal("undo did not publish restored task")
	}
}
