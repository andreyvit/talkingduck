package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHistoryPushRestartAndReopen(t *testing.T) {
	d := repo(t)
	state := filepath.Join(t.TempDir(), "state.json")
	opts := Options{Paths: []string{d}, NoAgents: true, HistoryPath: state}
	build := func() *Snapshot {
		t.Helper()
		s, e := Build(opts)
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	write(t, d, ".spec/pulled.md", "# Pulled task\n")
	command(t, d, "update-ref", "refs/remotes/origin/main", commit(t, d, "Other checkout task"))
	if got := build().Details.Repositories[0].Specs; len(got) != 0 {
		t.Fatalf("pulled-only: %+v", got)
	}
	write(t, d, ".spec/local.md", "---\nstage: FIN\nstage_complete: true\n---\n# Local task\n")
	h := commit(t, d, "Complete local task")
	if got := build().Details.Repositories[0].Specs; len(got) != 1 || got[0].RecentlyPushed {
		t.Fatalf("local: %+v", got)
	}
	command(t, d, "update-ref", "refs/remotes/origin/main", h)
	// Each Build reloads the file, as a restarted CLI or app would.
	s := build()
	if got := s.Details.Repositories[0].Specs; len(got) != 1 || !got[0].RecentlyPushed || got[0].Path != ".spec/local.md" {
		t.Fatalf("pushed: %+v", got)
	}
	task := objects(objects(s.Overview["repositories"])[0]["specs"])[0]
	if str(obj(task["workflow"])["stage"]) != "PUSHED" {
		t.Fatalf("workflow: %+v", task)
	}
	refreshLiveSnapshot(s, &liveState{sessions: map[string]*liveSession{}}, time.Now())
	if str(obj(task["workflow"])["stage"]) != "PUSHED" {
		t.Fatal("live refresh lost pushed state")
	}
	write(t, d, ".spec/local.md", "# Local task reopened\n")
	if got := build().Details.Repositories[0].Specs; len(got) != 1 || got[0].RecentlyPushed {
		t.Fatalf("reopened: %+v", got)
	}
	command(t, d, "checkout", "--", ".spec/local.md")
	if e := os.Remove(filepath.Join(d, ".spec/local.md")); e != nil {
		t.Fatal(e)
	}
	for _, s := range build().Details.Repositories[0].Specs {
		if s.RecentlyPushed {
			t.Fatal("deleted spec retained")
		}
	}
}

func TestHistoryBoundAndLatest(t *testing.T) {
	d := t.TempDir()
	state := filepath.Join(t.TempDir(), "state.json")
	now := time.Now().UTC()
	r := Repo{Path: d}
	for i := 0; i < 105; i++ {
		p := fmt.Sprintf(".spec/task%03d.md", i)
		write(t, d, p, "---\nstage: FIN\nstage_complete: true\n---\n# Task\n")
		r.Specs = append(r.Specs, &Spec{Path: p, LastActivity: now.Add(time.Duration(i) * time.Second)})
	}
	report := Report{ObservedAt: now, Repositories: []Repo{r}}
	if e := applyHistory(&report, state); e != nil {
		t.Fatal(e)
	}
	var saved inventoryHistory
	b, _ := os.ReadFile(state)
	if e := json.Unmarshal(b, &saved); e != nil {
		t.Fatal(e)
	}
	if len(saved.Worktrees[d].Specs) != 100 {
		t.Fatal("history not bounded")
	}
	// Active tasks stay first, followed by exactly one newest historical task.
	report = Report{ObservedAt: now.Add(time.Hour), Repositories: []Repo{{Path: d, Specs: []*Spec{{Path: ".spec/task000.md"}}}}}
	if e := applyHistory(&report, state); e != nil {
		t.Fatal(e)
	}
	specs := report.Repositories[0].Specs
	if len(specs) != 2 || specs[1].Path != ".spec/task104.md" || !specs[1].RecentlyPushed {
		t.Fatalf("selection: %+v", specs)
	}
	b, _ = os.ReadFile(state)
	_ = json.Unmarshal(b, &saved)
	for _, s := range saved.Worktrees[d].Specs {
		if s.Path == ".spec/task104.md" && !s.LastSeen.Equal(now) {
			t.Fatal("retention renewed historical last-seen")
		}
	}
	// Uncertain scans cannot turn missing tasks into pushed rows.
	report = Report{ObservedAt: now, Repositories: []Repo{{Path: d, Warnings: []string{"remote history unknown"}}}}
	if e := applyHistory(&report, state); e != nil {
		t.Fatal(e)
	}
	if len(report.Repositories[0].Specs) != 0 {
		t.Fatal("uncertain scan inferred push")
	}
}

func TestHistoryCorruptionPreserved(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	write(t, filepath.Dir(state), "state.json", "broken")
	if e := applyHistory(&Report{}, state); e == nil || !strings.Contains(e.Error(), "preserved") {
		t.Fatalf("error: %v", e)
	}
	b, _ := os.ReadFile(state)
	if string(b) != "broken" {
		t.Fatal("corrupt file overwritten")
	}
}

func TestHistoryConcurrentWorktrees(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		d := t.TempDir()
		write(t, d, ".spec/task.md", "# Task\n")
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := Report{ObservedAt: time.Now(), Repositories: []Repo{{Path: d, Specs: []*Spec{{Path: ".spec/task.md"}}}}}
			if e := applyHistory(&r, state); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	var saved inventoryHistory
	b, e := os.ReadFile(state)
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &saved); e != nil {
		t.Fatal(e)
	}
	if len(saved.Worktrees) != 8 {
		t.Fatalf("lost concurrent worktree: %d", len(saved.Worktrees))
	}
}

func TestHistorySkipsUnfinishedLegacyEntries(t *testing.T) {
	for _, body := range []string{
		"# Proposal\nNot changing code yet.\n## Research progress\n- [x] Inventory settings\n",
		"---\nstage: HL\nstage_complete: true\n---\n# Planned\n",
		"---\nstage: FIN\nstage_complete: false\nstatus: done\n---\n# Unfinished\n",
		"---\nstage: FIN\nstage: HL\nstage_complete: true\nstatus: done\n---\n",
	} {
		t.Run(body, func(t *testing.T) {
			d := t.TempDir()
			state := filepath.Join(t.TempDir(), "state.json")
			write(t, d, ".spec/proposal.md", body)
			write(t, d, ".spec/done.md", "---\nstatus: done\n---\n# Completed\n")
			now := time.Now().UTC()
			// Seed the old format: previously seen is the only recorded evidence.
			err := updateHistory(state, func(h *inventoryHistory) error {
				h.Worktrees[d] = worktreeHistory{Specs: []seenSpec{
					{Path: ".spec/proposal.md", LastSeen: now},
					{Path: ".spec/done.md", LastSeen: now.Add(-time.Hour)},
				}}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			report := Report{ObservedAt: now, Repositories: []Repo{{Path: d}}}
			if err := applyHistory(&report, state); err != nil {
				t.Fatal(err)
			}
			got := report.Repositories[0].Specs
			if len(got) != 1 || got[0].Path != ".spec/done.md" || !got[0].RecentlyPushed {
				t.Fatalf("expected older completed task, got %+v", got)
			}
		})
	}
}
