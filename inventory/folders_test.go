package inventory

import (
	"path/filepath"
	"testing"
)

func TestFolderMembershipDuringScan(t *testing.T) {
	path := repo(t)
	c, err := New(Options{Paths: []string{path}, NoAgents: true})
	if err != nil {
		t.Fatal(err)
	}
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	c.build = func(o Options) (*Snapshot, error) { close(started); <-release; return Build(o) }
	go func() { done <- c.Update(path) }()
	<-started
	if err = c.SetPaths(nil); err != nil {
		t.Fatal(err)
	}
	if len(c.Snapshot().Details.Repositories) != 0 {
		t.Fatal("hidden row visible during scan")
	}
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if len(c.Snapshot().Details.Repositories) != 0 {
		t.Fatal("scan resurrected hidden folder")
	}
	if err = c.Update(path); err == nil {
		t.Fatal("hidden folder scanned")
	}
	if _, err = c.UpdateLive(path); err == nil {
		t.Fatal("hidden folder updated")
	}
	if err = c.SetPaths([]string{path, path}); err != nil {
		t.Fatal(err)
	}
	if len(c.Paths()) != 1 || c.Snapshot().Details.Repositories[0].Cache.State != "ready" {
		t.Fatal("lost cached state")
	}
}
func TestMetadataFolderDiscovery(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("GROK_HOME", filepath.Join(home, ".grok"))
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	folder := repo(t)
	writeJSON(t, filepath.Join(home, ".claude", "sessions", "1.json"), Object{"cwd": folder})
	writeJSON(t, filepath.Join(home, ".grok", "sessions", "a", "b", "summary.json"), Object{"info": Object{"cwd": folder}})
	writeJSON(t, filepath.Join(home, ".claude", "sessions", "2.json"), Object{"cwd": t.TempDir()})
	paths := DiscoverFolders(home)
	if len(paths) != 1 || paths[0] != resolved(folder) {
		t.Fatal(paths)
	}
}
