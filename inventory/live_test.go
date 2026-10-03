package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIncrementalLogCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.jsonl")
	os.WriteFile(path, []byte("{\"n\":1}\n{\"n\":"), 0600)
	var c logCursor
	rows, reset, e := c.read(path)
	if e != nil || !reset || len(rows) != 1 {
		t.Fatal(rows, reset, e)
	}
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	f.WriteString("2}\n")
	f.Close()
	rows, reset, e = c.read(path)
	if e != nil || reset || len(rows) != 1 || number(rows[0]["n"]) != 2 {
		t.Fatal(rows, reset, e)
	}
	if rows, _, _ := c.read(path); len(rows) != 0 {
		t.Fatal(rows)
	}
	os.WriteFile(path, []byte("{\"n\":3}\n"), 0600)
	rows, reset, e = c.read(path)
	if e != nil || !reset || len(rows) != 1 || number(rows[0]["n"]) != 3 {
		t.Fatal(rows, reset, e)
	}
}
func TestLiveSpecRefreshDuringFullScan(t *testing.T) {
	dir := repo(t)
	path := ".spec/task.md"
	write(t, dir, path, "---\nrevision_round: 0\nstage: EX\nstage_complete: false\n---\n# Example task\n")
	c, e := New(Options{Paths: []string{dir}, NoAgents: true})
	if e != nil {
		t.Fatal(e)
	}
	if e = c.Update(dir); e != nil {
		t.Fatal(e)
	}
	baseline := c.entries[dir].snapshot
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	c.build = func(Options) (*Snapshot, error) {
		close(started)
		<-release
		b, _ := json.Marshal(baseline)
		var s Snapshot
		json.Unmarshal(b, &s)
		return &s, nil
	}
	go func() { done <- c.Update(dir) }()
	<-started
	write(t, dir, path, "---\nrevision_round: 0\nstage: FIN\nstage_complete: true\n---\n# Example task\n")
	t.Setenv("PATH", t.TempDir()) // Fast path and final reconciliation cannot invoke Git/ps.
	liveDone := make(chan error, 1)
	go func() {
		ok, e := c.UpdateLive(dir)
		if !ok && e == nil {
			t.Error("live update skipped during full scan")
		}
		liveDone <- e
	}()
	select {
	case e = <-liveDone:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		close(release)
		t.Fatal("blocked behind full scan")
	}
	check := func() {
		t.Helper()
		w := obj(objects(objects(c.Snapshot().Overview["repositories"])[0]["specs"])[0]["workflow"])
		if w["stage"] != "FIN" || w["progress"] != "✓" {
			t.Fatal(w)
		}
	}
	check()
	close(release)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	check()
	if c.Snapshot().Details.Repositories[0].Cache.ObservedAt.IsZero() {
		t.Fatal("Git observation lost")
	}
}
func TestIncrementalSessionFinishesTurn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	writeJSON(t, path, event("event_msg", Object{"type": "task_started"}, testNow))
	s := testSession("codex")
	s["transcript_path"] = path
	live := &liveState{sessions: map[string]*liveSession{}}
	now := time.Unix(int64(testNow), 0)
	live.update(s, now)
	if obj(s["activity"])["value"] != "running" {
		t.Fatal(s)
	}
	b, _ := json.Marshal(event("event_msg", Object{"type": "task_complete"}, testNow+1))
	f, _ := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	f.Write(append(b, '\n'))
	f.Close()
	live.update(s, now.Add(time.Second))
	if obj(s["activity"])["value"] != "idle" {
		t.Fatal(s)
	}
}

func TestLiveGrokLifecycleAndReceipt(t *testing.T) {
	dir := t.TempDir()
	chat := filepath.Join(dir, "chat_history.jsonl")
	writeJSON(t, filepath.Join(dir, "updates.jsonl"), update("agent_message_chunk", Object{"content": Object{"type": "text", "text": "Working"}}))
	writeJSON(t, filepath.Join(dir, "events.jsonl"), Object{"type": "turn_started", "ts": iso(testNow)})
	writeJSON(t, chat, Object{"type": "assistant", "content": "Zoo squash finished (.spec/task.md) r0"})
	s := testSession("grok")
	s["transcript_path"] = chat
	live := &liveState{sessions: map[string]*liveSession{}}
	now := time.Unix(int64(testNow), 0)
	live.update(s, now)
	if obj(s["activity"])["value"] != "running" || len(objects(s["zoo_receipts"])) != 1 {
		t.Fatal(s)
	}
	writeJSON(t, filepath.Join(dir, "events.jsonl"), Object{"type": "turn_ended", "ts": iso(testNow + 1)})
	live.update(s, now.Add(time.Second))
	if obj(s["activity"])["value"] != "idle" {
		t.Fatal(s)
	}
}
