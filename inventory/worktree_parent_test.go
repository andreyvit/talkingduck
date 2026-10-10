package inventory

import (
	"path/filepath"
	"testing"
)

func TestPendingWorktreeHasParent(t *testing.T) {
	root := repo(t)
	child := filepath.Join(t.TempDir(), "child")
	command(t, root, "worktree", "add", "--detach", child)
	cache, err := New(Options{Paths: []string{root, child}, NoAgents: true})
	if err != nil {
		t.Fatal(err)
	}
	rows := objects(cache.Snapshot().Overview["repositories"])
	want, _ := filepath.EvalSymlinks(root)
	if str(rows[1]["parent_path"]) != want {
		t.Fatalf("pending parent: %+v", rows[1])
	}
}
