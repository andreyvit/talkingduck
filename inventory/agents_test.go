package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const testNow = 1800000000.0

func testSession(engine string) Object {
	return newSession(engine, "session", "/synthetic/repo", "Test", "fixture", testNow, "desktop")
}
func event(typ string, p Object, at float64) Object {
	return Object{"type": typ, "timestamp": iso(at), "payload": p}
}
func update(kind string, fields Object) Object {
	fields["sessionUpdate"] = kind
	return Object{"timestamp": testNow + 500, "params": Object{"_meta": Object{"agentTimestampMs": testNow * 1000}, "update": fields}}
}
func writeJSON(t *testing.T, p string, v any) {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(p, append(b, '\n'), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestQuestionLifecycles(t *testing.T) {
	s := testSession("codex")
	rows := []Object{event("event_msg", Object{"type": "task_started"}, testNow), event("response_item", Object{"type": "function_call", "name": "request_user_input", "call_id": "q", "arguments": `{"questions":[{"question":"Which mode?"}]}`}, testNow)}
	st := codexTranscript(s, rows)
	finishQuestion(s, st.pending, "fixture", st.latest, true, false, true)
	if str(obj(s["question"])["value"]) != "yes" || str(obj(s["activity"])["value"]) != "waiting" {
		t.Fatal(s)
	}
	rows = append(rows, event("response_item", Object{"type": "function_call_output", "call_id": "q"}, testNow))
	if len(codexTranscript(s, rows).pending) != 0 {
		t.Fatal("answer not paired")
	}
	rows = append(rows, event("event_msg", Object{"type": "turn_aborted"}, testNow))
	if codexTranscript(s, rows).lifecycle != "interrupted" {
		t.Fatal("aborted turn")
	}
	async := []Object{event("response_item", Object{"type": "function_call", "name": "request_user_input_async", "call_id": "q", "arguments": "{}"}, testNow), event("response_item", Object{"type": "function_call_output", "call_id": "q"}, testNow)}
	s = testSession("codex")
	st = codexTranscript(s, async)
	finishQuestion(s, st.pending, "fixture", testNow, true, false, true)
	if str(obj(s["question"])["value"]) != "unknown" || str(obj(s["activity"])["value"]) == "waiting" {
		t.Fatal("async return incorrectly treated as answer/block")
	}
	s = testSession("claude")
	rows = []Object{{"type": "assistant", "timestamp": iso(testNow), "message": Object{"content": []Object{{"type": "tool_use", "id": "q", "name": "AskUserQuestion", "input": Object{"questions": []Object{{"question": "Approve?"}}}}}}}}
	st = claudeTranscript(s, rows)
	if len(st.pending) != 1 {
		t.Fatal("Claude question missing")
	}
	finishQuestion(s, st.pending, "fixture", testNow, false, false, true)
	if str(obj(s["question"])["value"]) != "unknown" {
		t.Fatal("old question treated as live")
	}
	rows = append(rows, Object{"type": "user", "message": Object{"content": []Object{{"type": "tool_result", "tool_use_id": "q"}}}})
	if len(claudeTranscript(s, rows).pending) != 0 {
		t.Fatal("Claude answer not paired")
	}
}
func TestGrokStreamsAndQuestionIDs(t *testing.T) {
	s := testSession("grok")
	rows := []Object{update("agent_message_chunk", Object{"content": Object{"type": "text", "text": "Hello "}}), update("agent_message_chunk", Object{"content": Object{"type": "text", "text": "world"}}), update("tool_call", Object{"toolCallId": "q", "title": "ask_user_question", "rawInput": Object{}}), update("tool_call_update", Object{"toolCallId": "q", "title": "Ask 3 questions"})}
	st := grokUpdates(s, rows)
	if len(st.pending) != 1 || str(obj(s["last_message"])["excerpt"]) != "Hello world" || timestamp(obj(s["last_message"])["at"]) != testNow {
		t.Fatal(s, st)
	}
	rows = append(rows, update("tool_call_update", Object{"toolCallId": "q", "status": "completed"}), update("turn_completed", Object{}))
	st = grokUpdates(s, rows)
	if len(st.pending) != 0 || !st.terminal {
		t.Fatal(st)
	}
}
func TestQuestionAtEndOfLongMessage(t *testing.T) {
	s := testSession("codex")
	message(s, "assistant", testNow, string(make([]byte, 600))+" Which option should I use?")
	finishQuestion(s, nil, "fixture", testNow, false, true, true)
	if str(obj(s["question"])["value"]) != "unknown" {
		t.Fatal(s)
	}
}
func TestTailBoundsPartialAndMalformed(t *testing.T) {
	p := filepath.Join(t.TempDir(), "events.jsonl")
	if e := os.WriteFile(p, []byte("{\"old\":true}\n{\"good\":true}\n{\"partial\":"), 0600); e != nil {
		t.Fatal(e)
	}
	rows, tr, bad, e := tailRows(p, 28)
	if e != nil || !tr || bad != 1 || len(rows) != 1 || rows[0]["good"] != true {
		t.Fatal(rows, tr, bad, e)
	}
}
func TestRecentSelectionKeepsLiveAndZeroMeansAll(t *testing.T) {
	older, newer := testSession("claude"), testSession("claude")
	older["id"], older["updated_at"] = "old", iso(testNow-100)
	newer["id"] = "new"
	for _, limit := range []int{0, 1} {
		if len(selectRecent([]Object{older, newer}, limit, map[string]bool{"old": true})) != 2 {
			t.Fatal(limit)
		}
	}
}
func TestSQLiteReadOnlyWALAndStaleCodexTurn(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "state_5.sqlite")
	writer, e := sql.Open("sqlite", path)
	if e != nil {
		t.Fatal(e)
	}
	defer writer.Close()
	for _, q := range []string{"PRAGMA journal_mode=WAL", "CREATE TABLE threads (id,cwd,title,name,updated_at,source,originator,rollout_path,archived)"} {
		if _, e = writer.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	log := filepath.Join(home, "rollout.jsonl")
	writeJSON(t, log, event("event_msg", Object{"type": "task_started"}, testNow-1000))
	_, e = writer.Exec("INSERT INTO threads VALUES (?,?,?,?,?,?,?,?,?)", "s", "/synthetic/repo", "raw", "Display name", testNow, "vscode", "codex_work_desktop", log, 0)
	if e != nil {
		t.Fatal(e)
	}
	before, _ := os.ReadFile(path)
	walBefore, _ := os.ReadFile(path + "-wal")
	reader, e := openReadDB(path)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, e = reader.ExecContext(ctx, "INSERT INTO threads(id) VALUES ('bad')"); e == nil {
		t.Fatal("read-only database accepted write")
	}
	reader.Close()
	ss, w, _ := collectCodex(home, map[string]bool{"/synthetic/repo": true}, 3, testNow)
	if len(ss) != 1 || len(w) != 0 || str(ss[0]["title"]) != "Display name" || str(obj(ss[0]["activity"])["value"]) != "unknown" {
		t.Fatal(ss, w)
	}
	after, _ := os.ReadFile(path)
	walAfter, _ := os.ReadFile(path + "-wal")
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(walBefore, walAfter) {
		t.Fatal("collector modified database/WAL")
	}
	missing := filepath.Join(home, "missing.sqlite")
	if db, e := openReadDB(missing); e == nil {
		db.Close()
		t.Fatal("missing database opened")
	}
	if exists(missing) {
		t.Fatal("created missing database")
	}
}
func TestClaudeNativeWaitAndMissingSources(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "sessions", "42.json")
	writeJSON(t, p, Object{"pid": 42, "sessionId": "s", "cwd": "/synthetic/repo", "entrypoint": "claude-desktop", "status": "waiting", "statusUpdatedAt": testNow * 1000})
	ss, _, _ := collectClaude(home, filepath.Join(home, "app"), map[string]bool{"/synthetic/repo": true}, 3, testNow, func(Object, string) Object { return finding("alive", "fixture", testNow, "observed", "") })
	if len(ss) != 1 || str(obj(ss[0]["activity"])["value"]) != "waiting" || str(obj(ss[0]["question"])["value"]) != "unknown" {
		t.Fatal(ss)
	}
	for _, key := range []string{"CODEX_HOME", "CLAUDE_CONFIG_DIR", "GROK_HOME"} {
		t.Setenv(key, "")
	}
	r := collectAgentData(t.TempDir(), []string{"/synthetic/repo"}, 0, testNow, probeProcess)
	if len(array(r["sessions"])) != 0 {
		t.Fatal(r)
	}
	for _, s := range array(r["sources"]) {
		if str(obj(s)["status"]) != "missing" {
			t.Fatal(r)
		}
	}
}
func TestGrokChildrenExcluded(t *testing.T) {
	home := t.TempDir()
	writeJSON(t, filepath.Join(home, "sessions", "repo", "child", "summary.json"), Object{"info": Object{"id": "child", "cwd": "/synthetic/repo"}, "session_kind": "subagent"})
	ss, _, count := collectGrok(home, nil, 3, testNow, probeProcess)
	if len(ss) != 0 || count != 0 {
		t.Fatal(ss, count)
	}
}

func TestGrokResumedSubagentRegistration(t *testing.T) {
	for _, tc := range []struct {
		name, process, registryCWD, parent string
		want                               bool
	}{
		{"resumed", "alive", "/synthetic/repo", "", true},
		{"stopped", "stopped", "/synthetic/repo", "", false},
		{"unverified", "unknown", "/synthetic/repo", "", false},
		{"wrong folder", "alive", "/other/repo", "", false},
		{"explicit child", "alive", "/synthetic/repo", "parent", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			folder := filepath.Join(home, "sessions", "repo", "session")
			writeJSON(t, filepath.Join(folder, "summary.json"), Object{"info": Object{"id": "session", "cwd": "/synthetic/repo"}, "session_kind": "subagent", "parent_session_id": tc.parent})
			writeJSON(t, filepath.Join(home, "active_sessions.json"), []Object{{"session_id": "session", "cwd": tc.registryCWD}})
			writeJSON(t, filepath.Join(folder, "events.jsonl"), Object{"type": "turn_started", "ts": iso(testNow)})
			ss, _, _ := collectGrok(home, nil, 3, testNow, func(Object, string) Object { return Object{"value": tc.process} })
			if (len(ss) == 1) != tc.want {
				t.Fatal(ss)
			}
			if tc.want {
				associate(ss[0], nil)
				if str(obj(ss[0]["activity"])["value"]) != "running" || str(obj(ss[0]["task_role"])["value"]) != "primary" {
					t.Fatal(ss[0])
				}
			}
		})
	}
}

func TestProcessProbeRejectsPIDReuse(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "ps")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if e := os.WriteFile(p, []byte("#!/bin/sh\nprintf 'Fri Jan  2 00:00:00 2026 /app/claude\\n'\n"), 0700); e != nil {
		t.Fatal(e)
	}
	r := Object{"pid": 42, "procStart": "Thu Jan  1 00:00:00 2026"}
	if str(probeProcess(r, "claude")["value"]) != "unknown" {
		t.Fatal("PID reuse accepted")
	}
	r["procStart"] = "Fri Jan  2 00:00:00 2026"
	if str(probeProcess(r, "claude")["value"]) != "alive" {
		t.Fatal("matching identity rejected")
	}
	if e := os.WriteFile(p, []byte("#!/bin/sh\nexit 1\n"), 0700); e != nil {
		t.Fatal(e)
	}
	if str(probeProcess(r, "claude")["value"]) != "stopped" {
		t.Fatal("missing process not detected")
	}
}
