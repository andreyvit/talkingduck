package telemetry

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Only metadata is read. Credential timestamps never establish recent use.
func recentlyUsed(id string, now time.Time) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	root := filepath.Join(home, "."+id)
	if id == "codex" && os.Getenv("CODEX_HOME") != "" {
		root = os.Getenv("CODEX_HOME")
	}
	if id == "claude" && os.Getenv("CLAUDE_CONFIG_DIR") != "" {
		root = os.Getenv("CLAUDE_CONFIG_DIR")
	}
	sub := "sessions"
	if id == "claude" {
		sub = "projects"
	}
	return recentFiles(filepath.Join(root, sub), id, now.Add(-14*24*time.Hour))
}
func recentFiles(root, id string, cutoff time.Time) bool {
	deadline := time.Now().Add(200 * time.Millisecond)
	budget := 4000
	var visit func(string, int) bool
	visit = func(dir string, depth int) bool {
		if depth > 6 || budget <= 0 || time.Now().After(deadline) {
			return false
		}
		// ReadDir(n) bounds memory even if a session folder has enormous history.
		f, err := os.Open(dir)
		if err != nil {
			return false
		}
		entries, _ := f.Readdir(budget)
		f.Close()
		budget -= len(entries)
		sort.Slice(entries, func(i, j int) bool { return entries[i].ModTime().After(entries[j].ModTime()) })
		for _, e := range entries {
			if time.Now().After(deadline) {
				return false
			}
			if e.Mode()&os.ModeSymlink != 0 {
				continue
			}
			name := e.Name()
			if e.IsDir() {
				if visit(filepath.Join(dir, name), depth+1) {
					return true
				}
				continue
			}
			if !e.Mode().IsRegular() {
				continue
			}
			valid := strings.HasSuffix(name, ".jsonl")
			if id == "grok" {
				valid = valid || name == "summary.json"
			}
			if valid && !e.ModTime().Before(cutoff) {
				return true
			}
		}
		return false
	}
	return visit(root, 0)
}
