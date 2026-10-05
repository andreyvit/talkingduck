package inventory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClaudeQuietToolWaitFullAndLive(t *testing.T) {
	home := t.TempDir()
	registry := filepath.Join(home, "sessions", "42.json")
	record := Object{"pid": 42, "procStart": "verified-start", "sessionId": "session", "cwd": "/synthetic/repo", "entrypoint": "claude-desktop", "status": "busy", "statusUpdatedAt": testNow - 900}
	writeJSON(t, registry, record)
	path := filepath.Join(home, "projects", "repo", "session.jsonl")
	writeJSON(t, path, Object{"type": "assistant", "timestamp": iso(testNow - 900), "message": Object{"stop_reason": "tool_use", "content": []Object{{"type": "tool_use", "name": "Agent", "id": "child", "input": Object{}}}}})
	sessions, _, _ := collectClaude(home, filepath.Join(home, "app"), nil, 3, testNow, func(Object, string) Object { return Object{"value": "alive"} })
	if len(sessions) != 1 {
		t.Fatal(sessions)
	}
	s := sessions[0]
	check := func(want string) {
		t.Helper()
		if obj(s["activity"])["value"] != want {
			t.Fatalf("want %s: %v", want, s["activity"])
		}
	}
	check("running")
	live := &liveState{sessions: map[string]*liveSession{}}
	now := time.Unix(int64(testNow), 0)
	live.update(s, now)
	check("running")
	record["status"] = "idle" // Parent is quiet while its child/tool still owns the turn.
	writeJSON(t, registry, record)
	live.update(s, now.Add(time.Minute))
	check("running")
	record["status"] = "waiting"
	writeJSON(t, registry, record)
	live.update(s, now.Add(2*time.Minute))
	check("waiting")
	record["status"] = "busy"
	writeJSON(t, registry, record)
	writeJSON(t, path, Object{"type": "assistant", "timestamp": iso(testNow), "message": Object{"stop_reason": "end_turn", "content": []Object{{"type": "text", "text": "Done"}}}})
	live.update(s, now.Add(3*time.Minute))
	check("idle")
	s["process"] = Object{"value": "stopped"}
	live.update(s, now.Add(4*time.Minute))
	check("stopped")
	record["pid"] = 43 // A replacement registration cannot borrow the old process probe.
	writeJSON(t, registry, record)
	s["process"] = Object{"value": "alive"}
	writeJSON(t, path, Object{"type": "assistant", "timestamp": iso(testNow + 1), "message": Object{"content": []Object{{"type": "tool_use", "name": "Agent", "id": "other"}}}})
	live.update(s, now.Add(5*time.Minute))
	check("unknown")
}

func TestCodexQuietOpenTurn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeJSON(t, path, event("event_msg", Object{"type": "task_started"}, testNow-1200))
	b, _ := json.Marshal(event("response_item", Object{"type": "function_call", "name": "wait_agent", "call_id": "wait", "arguments": "{}"}, testNow-1199))
	f, e := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	f.Write(append(b, '\n'))
	f.Close()
	s := testSession("codex")
	s["transcript_path"] = path
	live := &liveState{sessions: map[string]*liveSession{}}
	live.update(s, time.Unix(int64(testNow), 0))
	if obj(s["activity"])["value"] != "running" {
		t.Fatal(s["activity"])
	}
	writeJSON(t, path, event("event_msg", Object{"type": "turn_aborted"}, testNow))
	live.update(s, time.Unix(int64(testNow+1), 0))
	if obj(s["activity"])["value"] != "idle" {
		t.Fatal(s["activity"])
	}
}

func TestCodexToolWaitEndsAndQuestionsStayWaiting(t *testing.T) {
	s := testSession("codex")
	start := event("event_msg", Object{"type": "task_started"}, testNow-1200)
	call := event("response_item", Object{"type": "function_call", "name": "wait_agent", "call_id": "child"}, testNow-1100)
	st := codexTranscript(s, []Object{start, call})
	if !codexActiveTurn(st, testNow) {
		t.Fatal("quiet tool lost running")
	}
	st = codexTranscript(s, []Object{event("response_item", Object{"type": "function_call_output", "call_id": "child"}, testNow-1000)}, st)
	if codexActiveTurn(st, testNow) {
		t.Fatal("old finished tool kept running")
	}
	question := event("response_item", Object{"type": "function_call", "name": "request_user_input", "call_id": "question", "arguments": `{"questions":[{"question":"Which option?"}]}`}, testNow-900)
	st = codexTranscript(s, []Object{question}, st)
	finishQuestion(s, st.pending, "fixture", st.latest, codexActiveTurn(st, testNow), false, true)
	if obj(s["activity"])["value"] != "waiting" {
		t.Fatal(s["activity"])
	}
}
