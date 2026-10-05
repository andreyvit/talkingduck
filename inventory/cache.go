package inventory

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// RefreshState describes one worktree, never the freshness of the whole cache.
// ObservedAt stays at the last successful collection start when refresh fails.
// UpdatedAt marks completion of the latest attempt; zero times mean never.
type RefreshState struct {
	LiveObservedAt time.Time `json:"live_observed_at,omitempty"`
	State          string    `json:"state"` // pending, updating, ready, error
	ObservedAt     time.Time `json:"observed_at"`
	AttemptedAt    time.Time `json:"attempted_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Error          string    `json:"error,omitempty"`
}

type cachedWorktree struct {
	liveMu   sync.Mutex // Only cheap refresh/publication; never held during full scans.
	live     *liveState
	update   sync.Mutex // Serialize updates of this worktree only.
	snapshot *Snapshot  // Immutable once published, protected by Inventory.mu.
	refresh  RefreshState
}

// Inventory holds independent cached worktrees. Its zero value is not usable;
// create it with New. It owns no timer, goroutine, disk cache, or app lifecycle.
// Update and Snapshot may be called concurrently. Do not copy an Inventory.
type Inventory struct {
	generation uint64
	mu         sync.RWMutex
	paths      []string
	entries    map[string]*cachedWorktree
	archives   map[string][]string
	options    Options
	build      func(Options) (*Snapshot, error)
}

// New validates configuration without scanning. Pending folders are visible in
// Snapshot immediately. Paths returns their stable order for a caller's scheduler.
func New(opts Options) (*Inventory, error) {
	paths, err := worktreePaths(opts)
	if err != nil {
		return nil, err
	}
	opts.Paths = nil
	archives, err := loadArchives(opts.HistoryPath)
	if err != nil {
		return nil, err
	}
	cache := &Inventory{archives: archives, paths: paths, entries: map[string]*cachedWorktree{}, options: opts, build: Build}
	for _, path := range paths {
		cache.entries[path] = &cachedWorktree{refresh: RefreshState{State: "pending"}}
	}
	return cache, nil
}

// Paths returns a caller-owned copy of the configured absolute worktree paths.
func (c *Inventory) Paths() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.paths...)
}

// Update scans just one configured folder and atomically publishes its result.
// Other folders retain their previous observations. During collection Snapshot
// remains available. Failures keep the last successful data and mark it stale.
// Provider partial coverage remains a successful scan with source warnings.
func (c *Inventory) Update(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	c.mu.RLock()
	entry, ok := c.entries[path]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("worktree is not configured: %s", path)
	}
	entry.update.Lock()
	defer entry.update.Unlock()
	if !c.IsMonitored(path) {
		return fmt.Errorf("worktree is not monitored: %s", path)
	}
	c.mu.Lock()
	c.generation++
	entry.refresh.State = "updating"
	entry.refresh.AttemptedAt = time.Now().UTC()
	c.mu.Unlock()
	opts := c.options
	opts.Paths = []string{path}
	snapshot, err := c.build(opts)
	if err == nil && snapshot.Failed() {
		err = fmt.Errorf("%s: %s", path, snapshot.Details.Repositories[0].Error)
	}
	entry.liveMu.Lock()
	defer entry.liveMu.Unlock()
	if err == nil {
		if entry.live == nil {
			entry.live = &liveState{sessions: map[string]*liveSession{}}
		}
		refreshLiveSnapshot(snapshot, entry.live, time.Now().UTC())
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	entry.refresh.UpdatedAt = time.Now().UTC()
	if err != nil {
		entry.refresh.State = "error"
		entry.refresh.Error = err.Error()
		return err
	}
	entry.snapshot = snapshot
	entry.refresh.State = "ready"
	entry.refresh.Error = ""
	entry.refresh.ObservedAt = snapshot.Details.ObservedAt
	return nil
}

// Snapshot returns a detached, caller-owned view without filesystem/process
// reads or waiting for a scan. Copying cached data costs proportional to its size.
// Per-worktree cache metadata is authoritative: top-level observed_at is only
// the newest successful collection start, not a promise that every row is fresh.
func (c *Inventory) Snapshot() *Snapshot {
	type observation struct {
		path     string
		snapshot *Snapshot
		refresh  RefreshState
	}
	c.mu.RLock()
	observations := make([]observation, 0, len(c.paths))
	generation := c.generation
	archives := make(map[string][]string, len(c.archives))
	for path, rows := range c.archives {
		archives[path] = append([]string(nil), rows...)
	}
	for _, p := range c.paths {
		entry := c.entries[p]
		observations = append(observations, observation{p, entry.snapshot, entry.refresh})
	}
	c.mu.RUnlock()
	report := Report{Repositories: []Repo{}, AgentSources: []json.RawMessage{}}
	repos := []Object{}
	for _, o := range observations {
		state := o.refresh
		var row Object
		var repo Repo
		if o.snapshot == nil {
			why := state.Error
			repo = Repo{Path: o.path, Error: why, AgentState: "unknown", Specs: []*Spec{}}
			row = Object{"path": o.path, "error": why, "git": Object{"error": why}, "specs": []Object{}, "unassigned_sessions": []Object{}}
		} else {
			repo = o.snapshot.Details.Repositories[0]
			// Copy the row before attaching cache metadata; nested data remains immutable
			// until the final detached copy below.
			row = Object{}
			for k, v := range objects(o.snapshot.Overview["repositories"])[0] {
				row[k] = v
			}
			report.Boundary = o.snapshot.Details.Boundary
			if report.ObservedAt.Before(o.snapshot.Details.ObservedAt) {
				report.ObservedAt = o.snapshot.Details.ObservedAt
			}
			report.AgentLimitations = unique(append(report.AgentLimitations, o.snapshot.Details.AgentLimitations...))
			for _, raw := range o.snapshot.Details.AgentSources {
				var source Object
				if json.Unmarshal(raw, &source) == nil {
					source["worktree"], source["observed_at"] = o.path, o.snapshot.Details.ObservedAt
					b, _ := json.Marshal(source)
					report.AgentSources = append(report.AgentSources, b)
				}
			}
			// Retain provider errors per worktree rather than implying global freshness.
			row["agent_error"] = o.snapshot.Details.AgentError
			row["agent_sources"] = o.snapshot.Details.AgentSources
			row["agent_limitations"] = o.snapshot.Details.AgentLimitations
		}
		repo.Cache = &state
		row["cache"] = state
		report.Repositories = append(report.Repositories, repo)
		repos = append(repos, row)
	}
	overview := Object{"generation": generation, "schema_version": 2, "observed_at": report.ObservedAt, "boundary": report.Boundary, "repositories": repos, "agent_sources": report.AgentSources, "agent_limitations": report.AgentLimitations, "cached": true}
	// All values originate from JSON-compatible builder output; serialization also
	// detaches nested slices/maps in both the typed and dynamic reports.
	b, err := json.Marshal(Snapshot{Overview: overview, Details: report})
	if err != nil {
		panic("inventory snapshot is not serializable: " + err.Error())
	}
	var result Snapshot
	if err = json.Unmarshal(b, &result); err != nil {
		panic("inventory snapshot cannot be decoded: " + err.Error())
	}
	filterArchivedSnapshot(&result, archives)
	return &result
}

// SetPaths changes visible membership without discarding cached observations or
// per-folder scan locks. In-flight work may finish, but hidden rows stay hidden.
func (c *Inventory) SetPaths(paths []string) error {
	paths, err := worktreePaths(Options{Paths: paths})
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, path := range paths {
		if c.entries[path] == nil {
			c.entries[path] = &cachedWorktree{refresh: RefreshState{State: "pending"}}
		}
	}
	c.paths = paths
	c.generation++
	return nil
}
func (c *Inventory) IsMonitored(path string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return contains(c.paths, path)
}
