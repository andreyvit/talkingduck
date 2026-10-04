package inventory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ProcessChanges compares identity-checked registry processes, not wall-clock
// observation timestamps. It reads no transcripts and does not scan Git.
type ProcessChanges struct {
	home     string
	roots    map[string]string
	previous map[string]string
	probe    processProbe
}

func NewProcessChanges(paths []string, home string) *ProcessChanges {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	p := &ProcessChanges{home: home, roots: map[string]string{}, probe: probeProcess}
	for _, path := range paths {
		p.roots[resolved(path)] = path
	}
	return p
}
func (p *ProcessChanges) Poll() []string {
	values := map[string][]string{}
	for _, path := range p.roots {
		values[path] = nil
	}
	add := func(d Object, engine string) {
		if path := p.roots[resolved(str(d["cwd"]))]; path != "" {
			result := p.probe(d, engine)
			identity := []any{engine, d["pid"], d["procStart"], d["opened_at"], result["value"], result["reason"]}
			b, _ := json.Marshal(identity)
			values[path] = append(values[path], string(b))
		}
	}
	claude := envPath("CLAUDE_CONFIG_DIR", filepath.Join(p.home, ".claude"))
	for _, path := range glob(filepath.Join(claude, "sessions", "*.json")) {
		if d, e := readObject(path); e == nil && str(d["entrypoint"]) == "claude-desktop" {
			add(d, "claude")
		}
	}
	grok := envPath("GROK_HOME", filepath.Join(p.home, ".grok"))
	if b, e := os.ReadFile(filepath.Join(grok, "active_sessions.json")); e == nil {
		var rows []Object
		if json.Unmarshal(b, &rows) == nil {
			for _, d := range rows {
				add(d, "grok")
			}
		}
	}
	next := map[string]string{}
	changed := []string{}
	for path, rows := range values {
		sort.Strings(rows)
		next[path] = strings.Join(rows, "\n")
		if p.previous != nil && next[path] != p.previous[path] {
			changed = append(changed, path)
		}
	}
	p.previous = next
	sort.Strings(changed)
	return changed
}
func (p *ProcessChanges) Run(ctx context.Context, emit func(ChangeEvent)) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		for _, path := range p.Poll() {
			emit(ChangeEvent{Path: path, Expensive: true})
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
