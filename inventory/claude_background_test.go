package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClaudeBackgroundTasksOutliveParentTurn(t *testing.T) {
	home := t.TempDir()
	registry := filepath.Join(home, "sessions", "42.json")
	record := Object{"pid": 42, "procStart": "verified-start", "sessionId": "session", "cwd": "/synthetic/repo", "entrypoint": "claude-desktop", "status": "idle", "statusUpdatedAt": testNow}
	writeJSON(t, registry, record)
	path := filepath.Join(home, "projects", "repo", "session.jsonl")
	appendRow := func(row Object) {
		t.Helper()
		b, _ := json.Marshal(row)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.Write(append(b, '\n'))
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	appendRow(Object{"type": "user", "timestamp": iso(testNow - 900), "toolUseResult": Object{"backgroundTaskId": "shell"}})
	appendRow(Object{"type": "user", "timestamp": iso(testNow - 899), "toolUseResult": Object{"status": "async_launched", "agentId": "child"}})
	// Force the cheap tail to omit launches; it must retain the full scan's evidence.
	for i := 0; i < 300; i++ {
		appendRow(Object{"type": "attachment", "content": strings.Repeat("x", 1000)})
	}
	appendRow(Object{"type": "assistant", "timestamp": iso(testNow - 800), "message": Object{"stop_reason": "end_turn", "content": "Review still running"}})
	sessions, _, _ := collectClaude(home, filepath.Join(home, "app"), nil, 3, testNow, func(Object, string) Object { return Object{"value": "alive"} })
	if len(sessions) != 1 {
		t.Fatal(sessions)
	}
	s := sessions[0]
	check := func(want string, count int) {
		t.Helper()
		if str(obj(s["activity"])["value"]) != want || len(obj(s["claude_background_tasks"])) != count {
			t.Fatal(s["activity"], s["claude_background_tasks"])
		}
	}
	check("running", 2)
	live := &liveState{sessions: map[string]*liveSession{}}
	now := time.Unix(int64(testNow), 0)
	live.update(s, now)
	check("running", 2)
	notification := func(id, status string) string {
		return "<task-notification>\n<task-id>" + id + "</task-id>\n<status>" + status + "</status>\n</task-notification>"
	}
	appendRow(Object{"type": "queue-operation", "operation": "enqueue", "timestamp": iso(testNow + 1), "content": notification("shell", "completed")})
	live.update(s, now.Add(time.Second))
	check("running", 1)
	record["status"] = "waiting"
	writeJSON(t, registry, record)
	live.update(s, now.Add(2*time.Second))
	check("waiting", 1)
	record["status"] = "idle"
	writeJSON(t, registry, record)
	appendRow(Object{"type": "queue-operation", "operation": "remove", "reason": "absorbed_mid_turn", "timestamp": iso(testNow + 3), "content": notification("child", "failed")})
	live.update(s, now.Add(3*time.Second))
	check("idle", 0)
	// Completed background tasks must stay completed on a full rescan too.
	sessions, _, _ = collectClaude(home, filepath.Join(home, "app"), nil, 3, testNow+4, func(Object, string) Object { return Object{"value": "alive"} })
	s = sessions[0]
	check("idle", 0)
}

func TestClaudeBackgroundRequiresCurrentProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	start := time.Unix(int64(testNow-100), 0).UTC().Format("Mon Jan _2 15:04:05 2006")
	writeJSON(t, path, Object{"sessionId": "session", "pid": 42, "procStart": start, "status": "idle"})
	s := testSession("claude")
	s["id"], s["claude_registry_path"], s["claude_registry_pid"], s["claude_registry_start"] = "session", path, 42, start
	s["process"] = Object{"value": "alive"}
	st := transcriptState{terminal: true, background: Object{"old": iso(testNow - 200)}, pending: map[string]Object{}}
	claudeActivity(s, st, "transcript", testNow, true)
	if str(obj(s["activity"])["value"]) == "running" {
		t.Fatal("old process task revived")
	}
	st.background = Object{"current": iso(testNow - 50)}
	for _, process := range []string{"stopped", "unknown"} {
		s["process"] = Object{"value": process}
		claudeActivity(s, st, "transcript", testNow, true)
		if str(obj(s["activity"])["value"]) == "running" {
			t.Fatal(process, s["activity"])
		}
	}
}
