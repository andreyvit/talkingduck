package inventory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type signalFile struct {
	stamp string
	data  Object
}
type signalDB struct {
	stamp string
	rows  []Object
}

// ChangeProbe detects hints, not inventory state. Poll from one goroutine.
// It reads only spec metadata and agent metadata/transcript file metadata.
// No Git commands, source-tree traversal, transcript parsing, hooks, or writes.
type ChangeProbe struct {
	paths    []string
	home     string
	agents   bool
	previous map[string]string
	files    map[string]signalFile
	dbs      map[string]signalDB
}

func NewChangeProbe(paths []string, home string, agents bool) *ChangeProbe {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return &ChangeProbe{paths: append([]string(nil), paths...), home: home, agents: agents, files: map[string]signalFile{}, dbs: map[string]signalDB{}}
}
func fileSignal(path string) string {
	s, e := os.Stat(path)
	if e != nil {
		return "missing"
	}
	return fmt.Sprintf("%d:%d:%s", s.ModTime().UnixNano(), s.Size(), s.Mode())
}
func (p *ChangeProbe) metadata(path string) Object {
	stamp := fileSignal(path)
	if old, ok := p.files[path]; ok && old.stamp == stamp {
		return old.data
	}
	data, e := readObject(path)
	if e != nil {
		return nil
	} // Retry partial concurrent writes next poll.
	p.files[path] = signalFile{stamp, data}
	return data
}

// Poll establishes a baseline on first call; later calls return changed folders.
// Errors are advisory: the normal full scan remains the recovery path.
func (p *ChangeProbe) Poll() ([]string, []string) {
	signals := map[string][]string{}
	roots := map[string]string{}
	for _, path := range p.paths {
		roots[resolved(path)] = path
		signals[path] = nil
	}
	add := func(cwd, path string) {
		if root := roots[resolved(cwd)]; root != "" {
			signals[root] = append(signals[root], path+":"+fileSignal(path))
		}
	}
	warnings := []string{}
	for _, root := range p.paths {
		err := filepath.WalkDir(filepath.Join(root, ".spec"), func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".md") {
				add(root, path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			warnings = append(warnings, root+": spec signal unavailable")
		}
	}
	if p.agents {
		grok := envPath("GROK_HOME", filepath.Join(p.home, ".grok"))
		claude := envPath("CLAUDE_CONFIG_DIR", filepath.Join(p.home, ".claude"))
		// Directory enumeration discovers newly created Claude transcripts; contents
		// stay unread, and only sessions mapped to configured folders are statted.
		claudeTranscripts := map[string][]string{}
		for _, path := range glob(filepath.Join(claude, "projects", "*", "*.jsonl")) {
			id := strings.TrimSuffix(filepath.Base(path), ".jsonl")
			claudeTranscripts[id] = append(claudeTranscripts[id], path)
		}
		if b, err := os.ReadFile(filepath.Join(grok, "active_sessions.json")); err == nil {
			var records []Object
			if json.Unmarshal(b, &records) == nil {
				for _, record := range records {
					if root := roots[resolved(str(record["cwd"]))]; root != "" {
						encoded, _ := json.Marshal(record)
						signals[root] = append(signals[root], "grok registry:"+string(encoded))
					}
				}
			}
		}
		patterns := []string{
			filepath.Join(grok, "sessions", "*", "*", "summary.json"),
			filepath.Join(claude, "sessions", "*.json"),
			filepath.Join(p.home, "Library", "Application Support", "Claude", "claude-code-sessions", "*", "*", "local_*.json"),
		}
		seen := map[string]bool{}
		for _, pattern := range patterns {
			for _, path := range glob(pattern) {
				seen[path] = true
				d := p.metadata(path)
				cwd := str(first(d["cwd"], obj(d["info"])["cwd"]))
				if cwd == "" || roots[resolved(cwd)] == "" {
					continue
				}
				add(cwd, path)
				for _, transcript := range claudeTranscripts[str(first(d["cliSessionId"], d["sessionId"]))] {
					add(cwd, transcript)
				}
				if filepath.Base(path) == "summary.json" {
					for _, name := range []string{"events.jsonl", "updates.jsonl", "chat_history.jsonl"} {
						add(cwd, filepath.Join(filepath.Dir(path), name))
					}
				}
			}
		}
		for path := range p.files {
			if !seen[path] {
				delete(p.files, path)
			}
		}
		codex := envPath("CODEX_HOME", filepath.Join(p.home, ".codex"))
		for _, path := range append(glob(filepath.Join(codex, "state_*.sqlite")), glob(filepath.Join(codex, "sqlite", "state_*.sqlite"))...) {
			stamp := fileSignal(path) + fileSignal(path+"-wal")
			cached, ok := p.dbs[path]
			if !ok || cached.stamp != stamp {
				db, e := openReadDB(path)
				if e != nil {
					warnings = append(warnings, "Codex change probe: "+e.Error())
					continue
				}
				// updated_at is the established baseline schema. Unknown schemas fall back
				// to periodic inventory collection rather than waking every project.
				rows, e := queryObjects(db, "SELECT id,cwd,updated_at,rollout_path FROM threads ORDER BY updated_at DESC,id")
				db.Close()
				if e != nil {
					warnings = append(warnings, "Codex change probe: "+e.Error())
					continue
				}
				cached = signalDB{stamp, rows}
				p.dbs[path] = cached
			}
			counts := map[string]int{}
			for _, row := range cached.rows {
				cwd := str(row["cwd"])
				root := roots[resolved(cwd)]
				if root == "" || counts[root] >= 20 {
					continue
				}
				counts[root]++
				signals[root] = append(signals[root], fmt.Sprintf("%s:%v:%v", path, row["id"], row["updated_at"]))
				if rollout := str(row["rollout_path"]); rollout != "" {
					add(cwd, rollout)
				}
			}
		}
	}
	next := map[string]string{}
	changed := []string{}
	for _, path := range p.paths {
		sort.Strings(signals[path])
		next[path] = fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(signals[path], "\n"))))
		if p.previous != nil && next[path] != p.previous[path] {
			changed = append(changed, path)
		}
	}
	p.previous = next
	return changed, warnings
}
