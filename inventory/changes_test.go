package inventory

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestChangeProbeScopesSpecChanges(t *testing.T) {
	home := t.TempDir()
	a, b := filepath.Join(home, "a"), filepath.Join(home, "b")
	os.MkdirAll(a, 0755)
	os.MkdirAll(b, 0755)
	p := NewChangeProbe([]string{a, b}, home, false)
	if changed, _ := p.Poll(); len(changed) != 0 {
		t.Fatal(changed)
	}
	write(t, a, "source.go", "ignored source edit")
	if changed, _ := p.Poll(); len(changed) != 0 {
		t.Fatal(changed)
	}
	write(t, a, ".spec/new.md", "# New task")
	if changed, _ := p.Poll(); len(changed) != 1 || changed[0] != a {
		t.Fatal(changed)
	}
	if changed, _ := p.Poll(); len(changed) != 0 {
		t.Fatal(changed)
	}
	os.Remove(filepath.Join(a, ".spec/new.md"))
	if changed, _ := p.Poll(); len(changed) != 1 || changed[0] != a {
		t.Fatal(changed)
	}
}
func TestChangeProbeAgentMetadataAndTranscripts(t *testing.T) {
	home := t.TempDir()
	a := filepath.Join(home, "a")
	grok := filepath.Join(home, "grok")
	t.Setenv("GROK_HOME", grok)
	t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "claude"))
	folder := filepath.Join(grok, "sessions", "repo", "chat")
	p := NewChangeProbe([]string{a}, home, true)
	p.Poll()
	writeJSON(t, filepath.Join(folder, "summary.json"), Object{"info": Object{"cwd": a}})
	if changed, _ := p.Poll(); len(changed) != 1 || changed[0] != a {
		t.Fatal(changed)
	}
	write(t, folder, "events.jsonl", "new event")
	if changed, _ := p.Poll(); len(changed) != 1 || changed[0] != a {
		t.Fatal(changed)
	}
	writeJSON(t, filepath.Join(grok, "sessions", "other", "chat", "summary.json"), Object{"info": Object{"cwd": "/unconfigured"}})
	if changed, _ := p.Poll(); len(changed) != 0 {
		t.Fatal(changed)
	}
}
func TestChangeProbeCodexWALIsolation(t *testing.T) {
	home := t.TempDir()
	a, b := filepath.Join(home, "a"), filepath.Join(home, "b")
	codex := filepath.Join(home, "codex")
	os.MkdirAll(codex, 0755)
	t.Setenv("CODEX_HOME", codex)
	t.Setenv("GROK_HOME", filepath.Join(home, "grok"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "claude"))
	db, e := sql.Open("sqlite", filepath.Join(codex, "state_1.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	for _, q := range []string{"PRAGMA journal_mode=WAL", "CREATE TABLE threads (id TEXT,cwd TEXT,updated_at INTEGER,rollout_path TEXT)"} {
		if _, e := db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	db.Exec("INSERT INTO threads VALUES ('a',?,1,''),('b',?,1,'')", a, b)
	p := NewChangeProbe([]string{a, b}, home, true)
	if _, w := p.Poll(); len(w) != 0 {
		t.Fatal(w)
	}
	db.Exec("UPDATE threads SET updated_at=2 WHERE id='b'")
	changed, w := p.Poll()
	if len(w) != 0 || len(changed) != 1 || changed[0] != b {
		t.Fatal(changed, w)
	}
	if changed, _ := p.Poll(); len(changed) != 0 {
		t.Fatal(changed)
	}
}

func TestChangeProbeClaudeTranscriptAndGrokRegistry(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, "repo")
	claude, grok := filepath.Join(home, "claude"), filepath.Join(home, "grok")
	t.Setenv("CLAUDE_CONFIG_DIR", claude)
	t.Setenv("GROK_HOME", grok)
	t.Setenv("CODEX_HOME", filepath.Join(home, "codex"))
	writeJSON(t, filepath.Join(claude, "sessions", "1.json"), Object{"cwd": root, "sessionId": "chat"})
	p := NewChangeProbe([]string{root}, home, true)
	p.Poll()
	write(t, claude, "projects/project/chat.jsonl", "new message")
	if c, _ := p.Poll(); len(c) != 1 || c[0] != root {
		t.Fatal(c)
	}
	writeJSON(t, filepath.Join(grok, "active_sessions.json"), []Object{{"cwd": root, "session_id": "grok-chat", "pid": 1}})
	if c, _ := p.Poll(); len(c) != 1 || c[0] != root {
		t.Fatal(c)
	}
	writeJSON(t, filepath.Join(grok, "active_sessions.json"), []Object{})
	if c, _ := p.Poll(); len(c) != 1 || c[0] != root {
		t.Fatal(c)
	}
}
