package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitRoots(t *testing.T) {
	plain := t.TempDir()
	if IsGit(plain) {
		t.Fatal("ordinary folder accepted")
	}
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v", out, err)
		}
	}
	repo := filepath.Join(plain, "repo")
	git("init", "-q", repo)
	if !IsGit(repo) {
		t.Fatal("checkout rejected")
	}
	nested := filepath.Join(repo, "nested")
	_ = os.Mkdir(nested, 0700)
	if IsGit(nested) {
		t.Fatal("nested folder accepted as root")
	}
	git("-C", repo, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Initial")
	linked := filepath.Join(plain, "linked")
	git("-C", repo, "worktree", "add", "--detach", linked)
	if !IsGit(linked) {
		t.Fatal("linked worktree rejected")
	}
	bare := filepath.Join(plain, "bare")
	git("init", "--bare", "-q", bare)
	if IsGit(bare) {
		t.Fatal("bare repo accepted")
	}
	_ = os.WriteFile(filepath.Join(plain, ".git"), []byte("gitdir: missing\n"), 0600)
	if IsGit(plain) {
		t.Fatal("broken gitdir accepted")
	}
}
