package worktree

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLinkedParentOutsideDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	child := filepath.Join(t.TempDir(), "long-child")
	git := func(args ...string) {
		t.Helper()
		if b, e := exec.Command("git", args...).CombinedOutput(); e != nil {
			t.Fatalf("%v: %s", e, b)
		}
	}
	git("init", "-q", root)
	git("-C", root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Initial")
	git("-C", root, "worktree", "add", "--detach", child)
	want, _ := filepath.EvalSymlinks(root)
	if got := Parent(child); got != want {
		t.Fatalf("parent %q want %q", got, want)
	}
	if Parent(root) != "" {
		t.Fatal("ordinary checkout got a parent")
	}
}
