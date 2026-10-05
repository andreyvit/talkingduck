// Package inventory collects local Git/spec/session evidence without terminal I/O.
package inventory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Options controls collection. Empty Paths scans nothing.
// AgentLimit zero includes all local sessions.
type Options struct {
	HistoryPath string // Empty disables persistent recently-pushed history.
	Paths       []string
	NoAgents    bool
	AgentLimit  int
}

// Snapshot keeps the full correlated hierarchy and the legacy detailed report.
// Overview is the schema-versioned JSON payload, including sources and unknowns.
// Each Build returns fresh data; callers own it after Build completes.
type Snapshot struct {
	Overview Object
	Details  Report
}

// Failed reports checkout failures. Their errors remain in the partial snapshot.
func (s *Snapshot) Failed() bool {
	for _, repo := range s.Details.Repositories {
		if repo.Error != "" || (repo.Cache != nil && repo.Cache.Error != "") {
			return true
		}
	}
	return false
}

// Build scans sequentially and returns observations, never printing or exiting.
// Invalid options/path resolution return errors; unavailable checkouts/providers
// remain in the snapshot so other worktrees are still visible. Reads are bounded
// by per-command/database timeouts; GUI callers should run Build off their UI thread.
func Build(opts Options) (*Snapshot, error) {
	paths, err := worktreePaths(opts)
	if err != nil {
		return nil, err
	}
	report := Report{ObservedAt: time.Now().UTC(), Boundary: "Unpushed = HEAD commits absent from local remote-tracking refs. No fetch; remote knowledge may be stale. Agent states include uncertainty; see JSON evidence.", Repositories: []Repo{}}
	for _, p := range paths {
		report.Repositories = append(report.Repositories, scan(p, report.ObservedAt))
	}
	if err := applyHistory(&report, opts.HistoryPath); err != nil {
		for i := range report.Repositories {
			report.Repositories[i].Warnings = append(report.Repositories[i].Warnings, "Task history: "+err.Error())
		}
	}
	if !opts.NoAgents && len(paths) > 0 {
		roots := []string{}
		for _, r := range report.Repositories {
			roots = append(roots, r.Path)
		}
		agents := collectAgents(roots, opts.AgentLimit)
		report.AgentSources, report.AgentLimitations, report.AgentError = agents.Sources, agents.Limitations, agents.Error
		for _, raw := range agents.Sessions {
			var a AgentSession
			if json.Unmarshal(raw, &a) != nil {
				continue
			}
			cwd := resolved(a.Cwd)
			for i := range report.Repositories {
				if resolved(report.Repositories[i].Path) == cwd {
					report.Repositories[i].Agents = append(report.Repositories[i].Agents, raw)
				}
			}
		}
	}
	b, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	var overview Object
	if err = json.Unmarshal(b, &overview); err != nil {
		return nil, err
	}
	return &Snapshot{Overview: enrich(overview), Details: report}, nil
}

// CollectSessions returns raw provider observations without Git/spec correlation.
// Empty home uses the current user's home; provider environment overrides apply.
func CollectSessions(home string, paths []string, limit int) (Object, error) {
	if limit < 0 {
		return nil, fmt.Errorf("agent limit must be nonnegative")
	}
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, err
		}
	}
	return collectAgentData(home, paths, limit, float64(time.Now().UnixNano())/1e9, probeProcess), nil
}

func worktreePaths(opts Options) ([]string, error) {
	if opts.AgentLimit < 0 {
		return nil, fmt.Errorf("agent limit must be nonnegative")
	}
	paths := append([]string(nil), opts.Paths...)

	for i, p := range paths {
		absolute, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		paths[i] = absolute
	}
	return unique(paths), nil
}
