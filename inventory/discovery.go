package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"talkingduck/worktree"
)

// DiscoverFolders reads provider metadata only: no transcripts, process probes,
// Git commands, or recursive checkout scans. Discovery never enables monitoring.
func DiscoverFolders(home string) []string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	seen := map[string]bool{}
	add := func(value any) {
		p := str(value)
		if filepath.IsAbs(p) && worktree.IsGit(p) {
			if st, e := os.Stat(p); e == nil && st.IsDir() {
				seen[resolved(p)] = true
			}
		}
	}
	claude := envPath("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	grok := envPath("GROK_HOME", filepath.Join(home, ".grok"))
	for _, pattern := range []string{filepath.Join(claude, "sessions", "*.json"), filepath.Join(home, "Library", "Application Support", "Claude", "claude-code-sessions", "*", "*", "local_*.json"), filepath.Join(grok, "sessions", "*", "*", "summary.json")} {
		for _, path := range glob(pattern) {
			if d, e := readObject(path); e == nil {
				add(first(d["cwd"], obj(d["info"])["cwd"]))
			}
		}
	}
	if b, e := os.ReadFile(filepath.Join(grok, "active_sessions.json")); e == nil {
		var rows []Object
		if json.Unmarshal(b, &rows) == nil {
			for _, r := range rows {
				add(r["cwd"])
			}
		}
	}
	codex := envPath("CODEX_HOME", filepath.Join(home, ".codex"))
	for _, path := range append(glob(filepath.Join(codex, "state_*.sqlite")), glob(filepath.Join(codex, "sqlite", "state_*.sqlite"))...) {
		db, e := openReadDB(path)
		if e != nil {
			continue
		}
		rows, e := queryObjects(db, "SELECT DISTINCT cwd FROM threads")
		db.Close()
		if e == nil {
			for _, r := range rows {
				add(r["cwd"])
			}
		}
	}
	paths := []string{}
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}
