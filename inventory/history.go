package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type seenSpec struct {
	Path     string    `json:"path"`
	LastSeen time.Time `json:"last_seen"`
	Activity time.Time `json:"activity"`
}
type worktreeHistory struct {
	Specs        []seenSpec           `json:"recent_specs"`
	ArchivedRows []string             `json:"archived_rows,omitempty"`
	ArchivedAt   map[string]time.Time `json:"archived_at,omitempty"`
}
type inventoryHistory struct {
	Version   int                        `json:"version"`
	Worktrees map[string]worktreeHistory `json:"worktrees"`
}

func DefaultHistoryPath() string {
	root, e := os.UserConfigDir()
	if e != nil {
		return ""
	}
	return filepath.Join(root, "TalkingDuck", "state.json")
}
func validHistorySpec(path string) bool {
	return !filepath.IsAbs(path) && filepath.Clean(path) == path && specPath(filepath.ToSlash(path)) && !strings.Contains(path, "../")
}

// History records local sightings; retention also requires explicit completion.
// This is a read/merge/atomic-replace transaction shared by CLI and desktop.
func applyHistory(report *Report, path string) error {
	return updateHistory(path, func(state *inventoryHistory) error {
		for i := range report.Repositories {
			r := &report.Repositories[i]
			archived := state.Worktrees[r.Path].ArchivedRows
			r.Specs = visibleSpecs(r.Specs, archived)
			r.HideSummary = contains(archived, "")
			if r.Error != "" || len(r.Warnings) > 0 {
				continue
			}
			previous := state.Worktrees[r.Path]
			active := map[string]bool{}
			seen := map[string]seenSpec{}
			for _, old := range previous.Specs {
				if validHistorySpec(old.Path) {
					if st, e := os.Stat(filepath.Join(r.Path, old.Path)); e == nil && st.Mode().IsRegular() {
						seen[old.Path] = old
					}
				}
			}
			for _, s := range r.Specs {
				if !validHistorySpec(s.Path) || s.Deleted {
					continue
				}
				active[s.Path] = true
				seen[s.Path] = seenSpec{s.Path, report.ObservedAt, s.LastActivity}
			}
			ordered := []seenSpec{}
			for _, s := range seen {
				ordered = append(ordered, s)
			}
			sort.Slice(ordered, func(i, j int) bool {
				if !ordered[i].LastSeen.Equal(ordered[j].LastSeen) {
					return ordered[i].LastSeen.After(ordered[j].LastSeen)
				}
				if !ordered[i].Activity.Equal(ordered[j].Activity) {
					return ordered[i].Activity.After(ordered[j].Activity)
				}
				return ordered[i].Path < ordered[j].Path
			})
			if len(ordered) > 100 {
				ordered = ordered[:100]
			}
			state.Worktrees[r.Path] = worktreeHistory{Specs: ordered, ArchivedRows: previous.ArchivedRows, ArchivedAt: previous.ArchivedAt}
			for _, old := range ordered {
				if active[old.Path] || contains(archived, old.Path) {
					continue
				}
				b, e := os.ReadFile(filepath.Join(r.Path, old.Path))
				if e != nil {
					continue
				}
				s := parseSpec(old.Path, string(b))
				if !completedHistorySpec(s) {
					continue
				}
				s.RecentlyPushed = true
				s.LastActivity = old.Activity
				s.Evidence = append(s.Evidence, Evidence{"Previously displayed local task; no longer selected by local Git evidence", old.LastSeen.Format(time.RFC3339Nano), true})
				r.Specs = append(r.Specs, s)
				break
			}
		}
		return nil
	})
}

// completedHistorySpec rejects planning/research checklists and stale legacy status
// when newer workflow metadata exists. Unknown completion is not completion.
func completedHistorySpec(s *Spec) bool {
	frontmatter := zooFrontmatter(s.body)
	if matches(`(?m)^(?:stage|stage_complete):`, frontmatter) {
		stage, stageOK := zooField(s.body, "stage")
		complete, completeOK := zooField(s.body, "stage_complete")
		return stageOK && completeOK && stage == "FIN" && complete == "true"
	}
	return s.Stage == "done"
}

// updateHistory serializes read/modify/write across app processes.
func updateHistory(path string, update func(*inventoryHistory) error) error {
	if path == "" {
		return nil
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	lock, e := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	deadline := time.Now().Add(time.Second)
	for {
		e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if e == nil {
			break
		}
		if e != syscall.EWOULDBLOCK && e != syscall.EAGAIN {
			return e
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("history lock busy")
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	state := inventoryHistory{Version: 1, Worktrees: map[string]worktreeHistory{}}
	b, e := os.ReadFile(path)
	if e == nil {
		if e = json.Unmarshal(b, &state); e != nil {
			return fmt.Errorf("invalid history; preserved file: %w", e)
		}
		if state.Version != 1 || state.Worktrees == nil {
			return fmt.Errorf("unsupported history; preserved file")
		}
	} else if !os.IsNotExist(e) {
		return e
	}

	if e = update(&state); e != nil {
		return e
	}
	b, e = json.MarshalIndent(state, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".state-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if _, e = f.Write(append(b, '\n')); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(name, path)
}
