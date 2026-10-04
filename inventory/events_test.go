package inventory

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGitWatchRootsLinkedWorktree(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "tree")
	gitdir := filepath.Join(home, "shared", "worktrees", "tree")
	write(t, root, ".git", "gitdir: ../shared/worktrees/tree\n")
	write(t, gitdir, "commondir", "../..\n")
	got := gitWatchRoots(root)
	if len(got) != 2 || filepath.Clean(got[0]) != gitdir || got[1] != filepath.Join(home, "shared") {
		t.Fatal(got)
	}
}
func TestNativeEventsRouteWorktreeAndExternalGit(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	home := t.TempDir()
	write(t, root, ".git", "gitdir: "+external+"\n")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan ChangeEvent, 128)
	errs := make(chan error, 8)
	done := make(chan struct{})
	go func() {
		defer close(done)
		WatchChanges(ctx, []string{root}, home, false, time.Hour, func(e ChangeEvent) { events <- e }, func(e error) { errs <- e })
	}()
	next := func(want func(ChangeEvent) bool) {
		t.Helper()
		timer := time.NewTimer(3 * time.Second)
		defer timer.Stop()
		for {
			select {
			case e := <-events:
				if want(e) {
					return
				}
			case e := <-errs:
				t.Fatal(e)
			case <-timer.C:
				t.Fatal("missing native event")
			}
		}
	}
	next(func(e ChangeEvent) bool { return e.Reconcile })
	write(t, root, ".spec/task.md", "# Task\n")
	next(func(e ChangeEvent) bool { return e.Path == root && e.Cheap && e.Expensive })
	write(t, external, "HEAD", "ref: refs/heads/test\n")
	next(func(e ChangeEvent) bool { return e.Path == root && e.Expensive && !e.Cheap })
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("watch did not close")
	}
}
func TestProcessChangesOnlyEmitStateChanges(t *testing.T) {
	home := t.TempDir()
	a, b := filepath.Join(home, "a"), filepath.Join(home, "b")
	grok, claude := filepath.Join(home, "grok"), filepath.Join(home, "claude")
	t.Setenv("GROK_HOME", grok)
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	writeJSON(t, filepath.Join(grok, "active_sessions.json"), []Object{{"cwd": a, "pid": 1}})
	writeJSON(t, filepath.Join(claude, "sessions", "one.json"), Object{"cwd": b, "pid": 2, "entrypoint": "claude-desktop"})
	p := NewProcessChanges([]string{a, b}, home)
	alive := true
	p.probe = func(d Object, engine string) Object {
		v := "alive"
		if !alive && engine == "grok" {
			v = "stopped"
		}
		return finding(v, "test", float64(time.Now().UnixNano()), "observed", "test")
	}
	if c := p.Poll(); len(c) != 0 {
		t.Fatal(c)
	}
	if c := p.Poll(); len(c) != 0 {
		t.Fatal("timestamps caused false changes", c)
	}
	alive = false
	if c := p.Poll(); len(c) != 1 || c[0] != a {
		t.Fatal(c)
	}
	if c := p.Poll(); len(c) != 0 {
		t.Fatal(c)
	}
	if e := os.Remove(filepath.Join(claude, "sessions", "one.json")); e != nil {
		t.Fatal(e)
	}
	if c := p.Poll(); len(c) != 1 || c[0] != b {
		t.Fatal(c)
	}
}
