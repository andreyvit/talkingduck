package inventory

import (
	"path/filepath"
	"testing"
	"time"
)

func TestQuietGrokTurnFullAndIncremental(t *testing.T) {
	for _, tc := range []struct{ name, process, phase, want string }{
		{"quiet tool", "alive", "tool_execution", "running"},
		{"quiet reasoning", "alive", "thinking", "running"},
		{"completed", "alive", "ended", "idle"},
		{"dead process", "stopped", "tool_execution", "stopped"},
		{"unverified process", "unknown", "tool_execution", "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			dir := filepath.Join(home, "sessions", "repo", "session")
			writeJSON(t, filepath.Join(dir, "summary.json"), Object{"info": Object{"id": "session", "cwd": "/synthetic/repo"}})
			writeJSON(t, filepath.Join(home, "active_sessions.json"), []Object{{"session_id": "session", "cwd": "/synthetic/repo", "pid": 42, "opened_at": iso(testNow - 100)}})
			writeJSON(t, filepath.Join(dir, "updates.jsonl"), update("agent_message_chunk", Object{"content": Object{"type": "text", "text": "Starting tool"}}))
			e := Object{"type": "phase_changed", "phase": tc.phase, "ts": iso(testNow + 1)}
			if tc.phase == "ended" {
				e = Object{"type": "turn_ended", "ts": iso(testNow + 1), "outcome": "completed"}
			}
			writeJSON(t, filepath.Join(dir, "events.jsonl"), e)
			writeJSON(t, filepath.Join(dir, "chat_history.jsonl"), Object{"type": "assistant", "content": "Working"})
			now := testNow + 600
			sessions, _, _ := collectGrok(home, nil, 0, now, func(Object, string) Object { return Object{"value": tc.process} })
			if len(sessions) != 1 {
				t.Fatal(sessions)
			}
			s := sessions[0]
			if got := str(obj(s["activity"])["value"]); got != tc.want {
				t.Fatalf("full %s, want %s", got, tc.want)
			}
			live := &liveState{sessions: map[string]*liveSession{}}
			for _, offset := range []float64{0, 600} {
				live.update(s, time.Unix(int64(now+offset), 0))
				if got := str(obj(s["activity"])["value"]); got != tc.want {
					t.Fatalf("incremental %s, want %s", got, tc.want)
				}
			}
			// A new terminal event must end even a very quiet active turn.
			if tc.want == "running" {
				writeJSON(t, filepath.Join(dir, "events.jsonl"), Object{"type": "turn_ended", "ts": iso(now + 601), "outcome": "completed"})
				live.update(s, time.Unix(int64(now+602), 0))
				if str(obj(s["activity"])["value"]) != "idle" {
					t.Fatal("completion did not close turn", s)
				}
			}
		})
	}
}
func TestGrokOldRegistrationCannotBorrowCurrentProcess(t *testing.T) {
	home := t.TempDir()
	records := []Object{}
	for i, id := range []string{"old", "current"} {
		dir := filepath.Join(home, "sessions", "repo", id)
		writeJSON(t, filepath.Join(dir, "summary.json"), Object{"info": Object{"id": id, "cwd": "/synthetic/repo"}})
		writeJSON(t, filepath.Join(dir, "events.jsonl"), Object{"type": "turn_started", "ts": iso(testNow)})
		records = append(records, Object{"session_id": id, "cwd": "/synthetic/repo", "pid": 42, "opened_at": iso(testNow - 100 + float64(i))})
	}
	writeJSON(t, filepath.Join(home, "active_sessions.json"), records)
	sessions, _, _ := collectGrok(home, nil, 0, testNow+600, func(Object, string) Object { return Object{"value": "alive"} })
	if len(sessions) != 2 {
		t.Fatal(sessions)
	}
	for _, s := range sessions {
		want := "unknown"
		if str(s["id"]) == "current" {
			want = "running"
		}
		if got := str(obj(s["activity"])["value"]); got != want {
			t.Fatalf("%s: %s, want %s", s["id"], got, want)
		}
	}
}
