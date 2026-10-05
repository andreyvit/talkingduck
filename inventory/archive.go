package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func visibleSpecs(specs []*Spec, archived []string) []*Spec {
	result := make([]*Spec, 0, len(specs))
	for _, spec := range specs {
		if !contains(archived, spec.Path) {
			result = append(result, spec)
		}
	}
	return result
}

func loadArchives(path string) (map[string][]string, error) {
	result := map[string][]string{}
	if path == "" {
		return result, nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	var state inventoryHistory
	if err = json.Unmarshal(b, &state); err != nil {
		return nil, fmt.Errorf("invalid history; preserved file: %w", err)
	}
	if state.Version != 1 || state.Worktrees == nil {
		return nil, fmt.Errorf("unsupported history; preserved file")
	}
	for folder, state := range state.Worktrees {
		result[folder] = state.ArchivedRows
	}
	return result, nil
}

// ArchiveRow hides a task (relative spec path) or the task-free row (empty spec).
// Tombstones are independent of bounded recent history and remain even if running.
func (c *Inventory) ArchiveRow(folder, spec string) error {
	folder, err := filepath.Abs(folder)
	if err != nil {
		return err
	}
	c.mu.RLock()
	entry, ok := c.entries[folder]
	c.mu.RUnlock()
	if !ok || !c.IsMonitored(folder) {
		return fmt.Errorf("worktree is not configured: %s", folder)
	}
	if spec != "" && !validHistorySpec(spec) {
		return fmt.Errorf("invalid spec path")
	}
	if c.options.HistoryPath == "" {
		return fmt.Errorf("archiving requires a state file")
	}
	c.mu.RLock()
	known := spec == ""
	if entry.snapshot != nil {
		for _, s := range entry.snapshot.Details.Repositories[0].Specs {
			known = known || s.Path == spec
		}
	}
	c.mu.RUnlock()
	if !known {
		return fmt.Errorf("task is no longer in this worktree")
	}
	var archived []string
	err = updateHistory(c.options.HistoryPath, func(state *inventoryHistory) error {
		history := state.Worktrees[folder]
		if !contains(history.ArchivedRows, spec) {
			history.ArchivedRows = append(history.ArchivedRows, spec)
		}
		state.Worktrees[folder] = history
		archived = append([]string(nil), history.ArchivedRows...)
		return nil
	})
	if err != nil {
		return err
	}
	c.mu.Lock()
	// Concurrent archive calls must not replace each other's local tombstones.
	c.archives[folder] = unique(append(c.archives[folder], archived...))
	c.generation++
	c.mu.Unlock()
	return nil
}

// Apply the latest local tombstones after copying cached snapshots: an in-flight
// full or cheap scan can never publish an archived row back into view.
func filterArchivedSnapshot(snapshot *Snapshot, archives map[string][]string) {
	for i := range snapshot.Details.Repositories {
		repo := &snapshot.Details.Repositories[i]
		repo.Specs = visibleSpecs(repo.Specs, archives[repo.Path])
		repo.HideSummary = repo.HideSummary || contains(archives[repo.Path], "")
	}
	for _, repo := range objects(snapshot.Overview["repositories"]) {
		archived := archives[str(repo["path"])]
		if contains(archived, "") {
			repo["hide_summary"] = true
		}
		specs := []Object{}
		for _, spec := range objects(repo["specs"]) {
			if !contains(archived, str(spec["path"])) {
				specs = append(specs, spec)
			}
		}
		repo["specs"] = specs
	}
}
