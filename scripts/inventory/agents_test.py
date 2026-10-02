import importlib.util
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("agents", Path(__file__).with_name("agents.py"))
a = importlib.util.module_from_spec(spec)
spec.loader.exec_module(a)
NOW = 1800000000


def session(engine="codex"):
    return a.new_session(engine, "session", "/synthetic/repo", "Test", "fixture", NOW)


def event(typ, payload, at=NOW):
    return dict(type=typ, timestamp=a.iso(at), payload=payload)


def update(kind, at=NOW, **values):
    return dict(timestamp=at + 500, params={"_meta": {"agentTimestampMs": at * 1000}, "update": dict(sessionUpdate=kind, **values)})


class AgentTests(unittest.TestCase):
    def test_codex_question_answer_and_terminal(self):
        rows = [event("event_msg", {"type": "task_started"}),
                event("response_item", {"type": "function_call", "name": "request_user_input", "call_id": "q", "arguments": json.dumps({"questions": [{"question": "Which mode?"}]})})]
        s = session()
        pending, life, at, start = a.codex_transcript(s, rows, "fixture", NOW)
        a.finish_question(s, pending, "fixture", at, live=True)
        self.assertEqual((s["question"]["value"], s["activity"]["value"]), ("yes", "waiting"))
        rows.append(event("response_item", {"type": "function_call_output", "call_id": "q", "output": "answer"}))
        self.assertFalse(a.codex_transcript(session(), rows, "fixture", NOW)[0])
        rows.append(event("event_msg", {"type": "turn_aborted"}))
        self.assertEqual(a.codex_transcript(session(), rows, "fixture", NOW)[1], "interrupted")

    def test_async_tool_return_is_not_an_answer_or_block(self):
        rows = [event("event_msg", {"type": "task_started"}),
                event("response_item", {"type": "function_call", "name": "request_user_input_async", "call_id": "q", "arguments": "{}"}),
                event("response_item", {"type": "function_call_output", "call_id": "q", "output": "queued"})]
        s = session()
        pending, _, at, _ = a.codex_transcript(s, rows, "fixture", NOW)
        a.finish_question(s, pending, "fixture", at, live=True)
        self.assertEqual(s["question"]["value"], "unknown")
        self.assertNotEqual(s["activity"]["value"], "waiting")

    def test_old_pending_request_is_not_live(self):
        s = session()
        q = a.question_call("AskUserQuestion", "q", {"questions": [{"question": "Ship?"}]}, NOW - 9999)
        a.finish_question(s, {"q": q}, "fixture", NOW - 9999)
        self.assertEqual(s["question"]["value"], "unknown")
        self.assertTrue(s["pending_questions"])

    def test_claude_question_answer_pairing(self):
        rows = [dict(type="assistant", timestamp=a.iso(NOW), message={"content": [{"type": "tool_use", "id": "q", "name": "AskUserQuestion", "input": {"questions": [{"question": "Approve?"}]}}]})]
        self.assertTrue(a.claude_transcript(session("claude"), rows)[0])
        rows.append(dict(type="user", timestamp=a.iso(NOW), message={"content": [{"type": "tool_result", "tool_use_id": "q", "content": "Approved"}]}))
        self.assertFalse(a.claude_transcript(session("claude"), rows)[0])

    def test_grok_question_answer_and_message_clock(self):
        s = session("grok")
        rows = [update("user_message_chunk", content={"type": "text", "text": "Plan"}),
                update("tool_call", toolCallId="q", title="ask_user_question", rawInput={"questions": [{"question": "Which option?"}]})]
        self.assertTrue(a.grok_updates(s, rows)[0])
        self.assertEqual(s["last_message"]["at"], a.iso(NOW))  # agent clock, not later replay timestamp
        rows.append(update("tool_call_update", toolCallId="q", title="Ask a question"))
        self.assertTrue(a.grok_updates(s, rows)[0])
        rows.append(update("tool_call_update", toolCallId="q", status="completed"))
        self.assertFalse(a.grok_updates(s, rows)[0])
        rows.append(update("turn_completed", stop_reason="end_turn"))
        self.assertTrue(a.grok_updates(s, rows)[1])

    def test_grok_chunks_form_a_message(self):
        s = session("grok")
        a.grok_updates(s, [update("agent_message_chunk", content={"type": "text", "text": "Hello "}), update("agent_message_chunk", content={"type": "text", "text": "world"})])
        self.assertEqual(s["last_message"]["excerpt"], "Hello world")

    def test_question_near_end_of_long_prose_is_not_missed(self):
        s = session()
        a.message(s, "assistant", NOW, "Report. " * 200 + "Which option should I use?")
        a.finish_question(s, {}, "fixture", NOW, terminal=True)
        self.assertEqual(s["question"]["value"], "unknown")

    def test_tail_truncation_and_partial_record(self):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d) / "events.jsonl"
            p.write_bytes(b'{"old":true}\n{"good":true}\n{"partial":')
            rows, truncated, malformed = a.tail_rows(p, 28)
            self.assertTrue(truncated)
            self.assertEqual(rows, [{"good": True}])
            self.assertEqual(malformed, 1)

    def test_pid_reuse_is_not_live(self):
        record = dict(pid=42, procStart="Thu Jan  1 00:00:00 2026")
        with patch.object(a.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, "Fri Jan  2 00:00:00 2026 /app/claude\n", "")):
            self.assertEqual(a.process_probe(record, "claude")["value"], "unknown")
        with patch.object(a.subprocess, "run", return_value=subprocess.CompletedProcess([], 1, "", "")):
            self.assertEqual(a.process_probe(record, "claude")["value"], "stopped")

    def test_recent_selection_keeps_old_live_session(self):
        older, newer = session(), session()
        older.update(id="older", updated_at=a.iso(NOW - 100))
        newer.update(id="newer")
        self.assertEqual(len(a.select_recent([older, newer], 1, {"older"})), 2)

    def test_codex_readonly_db_and_stale_open_turn(self):
        with tempfile.TemporaryDirectory() as d:
            home = Path(d)
            log = home / "rollout.jsonl"
            log.write_text(json.dumps(event("event_msg", {"type": "task_started"}, NOW - 1000)) + "\n")
            db = home / "state_5.sqlite"
            c = sqlite3.connect(db)
            c.execute("CREATE TABLE threads (id,cwd,title,name,updated_at,source,originator,rollout_path,archived)")
            c.execute("INSERT INTO threads VALUES (?,?,?,?,?,?,?,?,?)", ("s", "/synthetic/repo", "raw prompt", "Display name", NOW, "vscode", "codex_work_desktop", str(log), 0))
            c.commit(); c.close()
            before = db.read_bytes()
            rows, warnings, _ = a.collect_codex(home, {"/synthetic/repo"}, 3, NOW)
            self.assertEqual(rows[0]["title"], "Display name")
            self.assertEqual(rows[0]["activity"]["value"], "unknown")
            self.assertEqual(db.read_bytes(), before)
            self.assertFalse(warnings)

    def test_claude_native_wait_without_question_is_not_question(self):
        with tempfile.TemporaryDirectory() as d:
            home = Path(d) / "cli"; app = Path(d) / "app"
            (home / "sessions").mkdir(parents=True)
            (home / "sessions/42.json").write_text(json.dumps(dict(pid=42, sessionId="s", cwd="/synthetic/repo", entrypoint="claude-desktop", status="waiting", statusUpdatedAt=NOW*1000)))
            rows, _, _ = a.collect_claude(home, app, {"/synthetic/repo"}, 3, NOW, lambda *args: a.finding("alive"))
            self.assertEqual(rows[0]["activity"]["value"], "waiting")
            self.assertEqual(rows[0]["question"]["value"], "unknown")
            self.assertEqual(rows[0]["permission"]["value"], "unknown")

    def test_grok_subagents_excluded(self):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d) / "sessions/repo/child"
            p.mkdir(parents=True)
            (p / "summary.json").write_text(json.dumps(dict(info=dict(id="child", cwd="/synthetic/repo"), session_kind="subagent")))
            rows, _, count = a.collect_grok(Path(d), {"/synthetic/repo"}, 3, NOW)
            self.assertFalse(rows)
            self.assertEqual(count, 0)

    def test_missing_sources_report_unknown_or_missing(self):
        with tempfile.TemporaryDirectory() as d:
            with patch.dict(os.environ, {}, clear=True):
                report = a.collect(Path(d), ["/synthetic/repo"], now=NOW)
            self.assertEqual([s["status"] for s in report["sources"]], ["missing"] * 3)
            self.assertFalse(report["sessions"])


if __name__ == "__main__":
    unittest.main()
