package inventory

import (
	"path/filepath"
	"testing"
)

func TestArchiveSurvivesScanRestartAndHistory(t *testing.T) {
	d := repo(t)
	state := filepath.Join(t.TempDir(), "state.json")
	write(t, d, ".spec/work.md", "---\nstatus: executing\n---\n# Work\n")
	opts := Options{Paths: []string{d}, NoAgents: true, HistoryPath: state}
	c, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Update(d); err != nil {
		t.Fatal(err)
	}
	if err = c.ArchiveRow(d, ".spec/work.md"); err != nil {
		t.Fatal(err)
	}
	assertHidden := func(s *Snapshot) {
		t.Helper()
		if len(s.Details.Repositories[0].Specs) != 0 || len(objects(objects(s.Overview["repositories"])[0]["specs"])) != 0 {
			t.Fatal("archived task visible")
		}
	}
	assertHidden(c.Snapshot())
	if err = c.Update(d); err != nil {
		t.Fatal(err)
	}
	assertHidden(c.Snapshot())
	h := commit(t, d, "finish task")
	command(t, d, "update-ref", "refs/remotes/origin/main", h)
	restarted, err := New(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err = restarted.Update(d); err != nil {
		t.Fatal(err)
	}
	assertHidden(restarted.Snapshot())
	// Later edits and an executing status do not override an explicit archive.
	write(t, d, ".spec/work.md", "---\nstatus: executing\n---\n# Work again\n")
	built, err := Build(opts)
	if err != nil {
		t.Fatal(err)
	}
	assertHidden(built)
	archived, err := loadArchives(state)
	if err != nil || !contains(archived[d], ".spec/work.md") {
		t.Fatal(archived, err)
	}
}

func TestArchiveWinsOverInflightScan(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/work.md", "# Work")
	c, err := New(Options{Paths: []string{d}, NoAgents: true, HistoryPath: filepath.Join(t.TempDir(), "state.json")})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Update(d); err != nil {
		t.Fatal(err)
	}
	stale := c.Snapshot()
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	c.build = func(Options) (*Snapshot, error) { close(started); <-release; return stale, nil }
	go func() { done <- c.Update(d) }()
	<-started
	if err = c.ArchiveRow(d, ".spec/work.md"); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	if len(c.Snapshot().Details.Repositories[0].Specs) != 0 {
		t.Fatal("scan resurrected archive")
	}
}

func TestArchiveSummaryAndValidation(t *testing.T) {
	d := repo(t)
	state := filepath.Join(t.TempDir(), "state.json")
	c, err := New(Options{Paths: []string{d}, NoAgents: true, HistoryPath: state})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.ArchiveRow(d, ""); err != nil {
		t.Fatal(err)
	}
	if !c.Snapshot().Details.Repositories[0].HideSummary {
		t.Fatal("summary still visible")
	}
	for _, spec := range []string{"../work.md", "/tmp/work.md", ".spec/unknown.md"} {
		if c.ArchiveRow(d, spec) == nil {
			t.Fatal("accepted invalid archive", spec)
		}
	}
	if c.ArchiveRow(filepath.Dir(d), "") == nil {
		t.Fatal("accepted unconfigured worktree")
	}
	write(t, d, ".spec/new.md", "# New")
	if err = c.Update(d); err != nil {
		t.Fatal(err)
	}
	if len(c.Snapshot().Details.Repositories[0].Specs) != 1 {
		t.Fatal("summary archive hid future tasks")
	}
	disabled, err := New(Options{Paths: []string{d}})
	if err != nil {
		t.Fatal(err)
	}
	if disabled.ArchiveRow(d, "") == nil {
		t.Fatal("archive pretended to persist without state file")
	}
}
