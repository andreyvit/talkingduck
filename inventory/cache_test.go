package inventory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestCacheIncrementalRefreshAndFailure(t *testing.T) {
	first, second := repo(t), repo(t)
	cache, err := New(Options{Paths: []string{first, second}, NoAgents: true})
	if err != nil {
		t.Fatal(err)
	}
	paths := cache.Paths()
	paths[0] = "caller mutation"
	if cache.Paths()[0] != first {
		t.Fatal("paths shared")
	}
	initial := cache.Snapshot()
	for _, r := range initial.Details.Repositories {
		if r.Cache.State != "pending" || !r.Cache.ObservedAt.IsZero() {
			t.Fatal(r)
		}
	}
	write(t, first, ".spec/a.md", "---\nstatus: planning\n---\n# A\n")
	if err = cache.Update(first); err != nil {
		t.Fatal(err)
	}
	current := cache.Snapshot()
	if current.Details.Repositories[0].Cache.State != "ready" || current.Details.Repositories[1].Cache.State != "pending" {
		t.Fatal(current)
	}
	if len(current.Details.Repositories[0].Specs) != 1 {
		t.Fatal(current)
	}
	// A caller can mutate both representations without corrupting the cache.
	current.Details.Repositories[0].Specs[0].Title = "mutated"
	objects(current.Overview["repositories"])[0]["path"] = "mutated"
	if fresh := cache.Snapshot(); fresh.Details.Repositories[0].Specs[0].Title == "mutated" || objects(fresh.Overview["repositories"])[0]["path"] != first {
		t.Fatal("shared cache data")
	}
	if err = cache.Update(second); err != nil {
		t.Fatal(err)
	}
	before := cache.Snapshot()
	secondBefore, _ := json.Marshal(objects(before.Overview["repositories"])[1])
	write(t, first, "new.go", "package sample")
	if err = cache.Update(first); err != nil {
		t.Fatal(err)
	}
	after := cache.Snapshot()
	secondAfter, _ := json.Marshal(objects(after.Overview["repositories"])[1])
	if string(secondBefore) != string(secondAfter) {
		t.Fatal("refresh changed another worktree")
	}
	if len(after.Details.Repositories[0].LocalChanges) != 2 {
		t.Fatal("refresh missed new path")
	}
	observed := after.Details.Repositories[0].Cache.ObservedAt
	if err = os.Rename(first, first+"-away"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Rename(first+"-away", first) })
	if err = cache.Update(first); err == nil {
		t.Fatal("missing worktree reported success")
	}
	stale := cache.Snapshot()
	state := stale.Details.Repositories[0].Cache
	if state.State != "error" || state.Error == "" || !state.ObservedAt.Equal(observed) || !stale.Failed() || len(stale.Details.Repositories[0].Specs) != 1 {
		t.Fatal(stale)
	}
	if err = os.Rename(first+"-away", first); err != nil {
		t.Fatal(err)
	}
	if err = cache.Update(first); err != nil {
		t.Fatal(err)
	}
	if result := cache.Snapshot(); result.Failed() || result.Details.Repositories[0].Cache.Error != "" {
		t.Fatal("error not cleared")
	}
}

func mockSnapshot(path string) *Snapshot {
	at := time.Now().UTC()
	return &Snapshot{Details: Report{ObservedAt: at, Repositories: []Repo{{Path: path, Specs: []*Spec{}}}}, Overview: Object{"repositories": []Object{{"path": path, "specs": []Object{}, "git": Object{"cleanliness": "CLEAN"}}}}}
}

func TestCacheReadsDoNotWaitForRefresh(t *testing.T) {
	first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
	cache, err := New(Options{Paths: []string{first, second}})
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	cache.build = func(opts Options) (*Snapshot, error) {
		if len(opts.Paths) != 1 {
			t.Error("scan not scoped to one folder")
		}
		if opts.Paths[0] == first {
			close(entered)
			<-release
		}
		return mockSnapshot(opts.Paths[0]), nil
	}
	done := make(chan error, 1)
	go func() { done <- cache.Update(first) }()
	<-entered
	read := make(chan *Snapshot, 1)
	go func() { read <- cache.Snapshot() }()
	select {
	case s := <-read:
		if s.Details.Repositories[0].Cache.State != "updating" {
			t.Fatal(s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("read waited for scan")
	}
	other := make(chan error, 1)
	go func() { other <- cache.Update(second) }()
	select {
	case err = <-other:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("other worktree blocked")
	}
	unblock()
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if cache.Snapshot().Failed() {
		t.Fatal("completed snapshot failed")
	}
}

func TestCacheSerializesSameWorktree(t *testing.T) {
	path := t.TempDir()
	cache, _ := New(Options{Paths: []string{path}})
	entered, release := make(chan int, 2), make(chan struct{})
	var count int
	cache.build = func(opts Options) (*Snapshot, error) {
		count++
		entered <- count
		if count == 1 {
			<-release
		}
		return mockSnapshot(path), nil
	}
	done := make(chan error, 2)
	go func() { done <- cache.Update(path) }()
	<-entered
	go func() { done <- cache.Update(path) }()
	// Snapshot remains available even with a second update queued.
	if cache.Snapshot().Details.Repositories[0].Cache.State != "updating" {
		t.Fatal("not refreshing")
	}
	close(release)
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if n := <-entered; n != 2 {
		t.Fatal(n)
	}
}

func TestCacheValidationAndFatalFailure(t *testing.T) {
	if _, err := New(Options{AgentLimit: -1}); err == nil {
		t.Fatal("invalid limit accepted")
	}
	path := t.TempDir()
	cache, err := New(Options{Paths: []string{path, filepath.Join(path, ".")}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cache.Paths(), []string{path}) {
		t.Fatal(cache.Paths())
	}
	calls := 0
	cache.build = func(Options) (*Snapshot, error) { calls++; return nil, errors.New("scan unavailable") }
	if err = cache.Update(filepath.Join(path, "other")); err == nil || calls != 0 {
		t.Fatal("unknown worktree scanned")
	}
	if err = cache.Update(path); err == nil {
		t.Fatal("failure hidden")
	}
	s := cache.Snapshot()
	state := s.Details.Repositories[0].Cache
	if state.State != "error" || state.Error != "scan unavailable" || !state.ObservedAt.IsZero() || !s.Failed() {
		t.Fatal(s)
	}
}

func TestCacheProviderCoverageStaysWorktreeScoped(t *testing.T) {
	first, second := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")
	cache, _ := New(Options{Paths: []string{first, second}})
	cache.build = func(opts Options) (*Snapshot, error) {
		s := mockSnapshot(opts.Paths[0])
		s.Details.AgentSources = []json.RawMessage{json.RawMessage(`{"engine":"grok","status":"partial","warnings":["missing transcript"]}`)}
		if opts.Paths[0] == first {
			s.Details.AgentError = "first-folder provider failure"
		}
		return s, nil
	}
	if err := cache.Update(first); err != nil {
		t.Fatal(err)
	}
	firstTime := cache.Snapshot().Details.Repositories[0].Cache.ObservedAt
	if err := cache.Update(second); err != nil {
		t.Fatal(err)
	}
	snapshot := cache.Snapshot()
	rows := objects(snapshot.Overview["repositories"])
	if rows[0]["agent_error"] != "first-folder provider failure" || rows[1]["agent_error"] != "" {
		t.Fatal(rows)
	}
	sources := objects(snapshot.Overview["agent_sources"])
	if len(sources) != 2 || sources[0]["worktree"] != first || sources[1]["worktree"] != second || !snapshot.Details.Repositories[0].Cache.ObservedAt.Equal(firstTime) {
		t.Fatal(sources)
	}
	if snapshot.Failed() {
		t.Fatal("partial provider coverage should remain a usable scan")
	}
}
