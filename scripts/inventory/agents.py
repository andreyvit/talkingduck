#!/usr/bin/env python3
"""Read-only agent inventory. Private formats are best effort, never live authority by age alone."""
import argparse
from contextlib import contextmanager
import datetime as dt
import json
import os
from pathlib import Path
import re
import sqlite3
import subprocess
import time

TAIL_BYTES = 4 * 1024 * 1024
FRESH_SECONDS = 120
QUESTION_TOOLS = {"request_user_input", "request_user_input_async", "AskUserQuestion", "ask_user_question"}


def stamp(value):
    if isinstance(value, (int, float)):
        return value / 1000 if value > 10**11 else float(value)
    if isinstance(value, str):
        try:
            return dt.datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp()
        except ValueError:
            pass
    return None


def iso(value):
    value = stamp(value)
    return dt.datetime.fromtimestamp(value, dt.timezone.utc).isoformat().replace("+00:00", "Z") if value else None


def read_json(path):
    with Path(path).open() as f:
        return json.load(f)


def tail_rows(path, budget=TAIL_BYTES):
    """Ignore an incomplete last record; surface malformed records and bounded history."""
    path = Path(path)
    with path.open("rb") as f:
        size = os.fstat(f.fileno()).st_size
        start = max(0, size - budget)
        f.seek(start)
        data = f.read(budget)
    if start:
        data = data.partition(b"\n")[2]
    rows, malformed = [], 0
    for line in data.splitlines(keepends=True):
        if not line.endswith(b"\n"):
            malformed += 1
            continue
        try:
            row = json.loads(line)
            if isinstance(row, dict):
                rows.append(row)
            else:
                malformed += 1
        except (ValueError, UnicodeError):
            malformed += 1
    return rows, start > 0, malformed


def finding(value="unknown", source=None, at=None, confidence="unknown", reason="Not observed"):
    return dict(value=value, source=str(source) if source else None, at=iso(at), confidence=confidence, reason=reason)


def new_session(engine, sid, cwd, title, source, updated=None, client="unknown"):
    return dict(engine=engine, client=client, id=sid, app_session_id=None, parent_id=None,
                cwd=cwd, title=title or sid, updated_at=iso(updated), source=str(source),
                last_message=dict(at=None, role=None, excerpt=None),
                activity=finding(), question=finding(), permission=finding(), process=finding(),
                spec_association=finding(reason="Folder match only; explicit spec ownership not established"),
                pending_questions=[], last_turn_status=None, unknown_fields=[], warnings=[])


def clean_text(content):
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(str(x.get("text", "")) for x in content if isinstance(x, dict) and x.get("type") in ("text", "input_text", "output_text"))
    return ""


def message(s, role, at, content):
    text = clean_text(content)
    if not text.strip():
        return
    # Excerpts are bounded; tools, reasoning, attachments, and command output are not exported.
    s["last_message"] = dict(at=iso(at), role=role, excerpt=text if len(text)<=400 else text[:240]+" … "+text[-160:])


def question_call(name, call_id, args, at):
    if name.split(".")[-1] not in QUESTION_TOOLS:
        return None
    if isinstance(args, str):
        try:
            args = json.loads(args)
        except ValueError:
            args = {}
    args = args if isinstance(args, dict) else {}
    qs = args.get("questions", [])
    prompts = []
    for q in qs if isinstance(qs, list) else []:
        if isinstance(q, dict):
            prompts.append(str(q.get("question", q.get("title", "")))[:1000])
    return dict(call_id=call_id, tool=name, at=iso(at), prompts=prompts,
                blocking=name.split(".")[-1] != "request_user_input_async")


def finish_question(s, pending, source, at, live=False, terminal=False, complete_tail=False):
    s["pending_questions"] = list(pending.values())
    if pending:
        s["question"] = finding("yes" if live else "unknown", source, at,
                                "observed" if live else "inferred",
                                "Unresolved structured question" + (" and live session evidence" if live else "; current wait is unverified"))
        if any(not q["blocking"] for q in pending.values()):
            s["question"] = finding("unknown", source, at, "inferred", "Async question request found; immediate tool return does not establish whether it was answered")
        if live and any(q["blocking"] for q in pending.values()):
            s["activity"] = finding("waiting", source, at, "inferred", "Unresolved blocking question in a live session")
        return
    excerpt = s["last_message"]["excerpt"] or ""
    if s["last_message"]["role"] == "assistant" and re.search(r"\?|please confirm|let me know|your approval", excerpt, re.I):
        s["question"] = finding("unknown", source, s["last_message"]["at"], "inferred", "Latest assistant text may ask a question; caller should read it")
    elif terminal or (complete_tail and live):
        s["question"] = finding("no", source, at, "inferred", "No unresolved structured question or question cue in the latest message")


def add_tail_warnings(s, path, truncated, malformed):
    if truncated:
        s["warnings"].append(f"Bounded tail of {Path(path).name}; older unmatched requests may be missing")
    if malformed:
        s["warnings"].append(f"Skipped {malformed} malformed/incomplete records in {Path(path).name}")


def process_probe(record, engine):
    pid = record.get("pid")
    if not isinstance(pid, int) or pid <= 0:
        return finding(reason="No process ID")
    try:
        env = dict(os.environ, TZ="UTC", LC_ALL="C")
        p = subprocess.run(["ps", "-p", str(pid), "-o", "lstart=,comm="], text=True, capture_output=True, timeout=3, env=env)
        if p.returncode == 1 and not p.stdout.strip():
            return finding("stopped", f"ps:{pid}", time.time(), "observed", "Registered process no longer exists")
        if p.returncode:
            return finding(reason=f"Process query failed for PID {pid}")
        line = p.stdout.strip()
        match = re.match(r"(.{24})\s+(.+)", line)
        if not match:
            return finding(reason="Unrecognized ps format")
        start, executable = match.groups()
        if engine == "claude":
            expected = record.get("procStart")
            if not expected or " ".join(start.split()) != " ".join(expected.split()) or "claude" not in executable.lower():
                return finding(reason="Claude process identity/start time did not match registry")
        else:
            opened = stamp(record.get("opened_at"))
            began = dt.datetime.strptime(start, "%a %b %d %H:%M:%S %Y").replace(tzinfo=dt.timezone.utc).timestamp()
            if not opened or began > opened + 2 or "grok" not in Path(executable).name.lower():
                return finding(reason="Grok process identity/start time did not match registry")
        return finding("alive", f"ps:{pid}", time.time(), "observed", "Executable and registered process lifetime match")
    except (OSError, ValueError, subprocess.TimeoutExpired):
        return finding(reason="Process check unavailable")


def scoped(cwd, roots):
    return isinstance(cwd, str) and (not roots or str(Path(cwd).resolve()) in roots)


def select_recent(sessions, limit, extra_ids=()):
    sessions.sort(key=lambda s: (stamp(s["updated_at"]) or 0, s["id"]), reverse=True)
    counts, selected = {}, []
    for s in sessions:
        key = s["cwd"]
        n = counts.get(key, 0)
        if limit == 0 or n < limit or s["id"] in extra_ids:
            selected.append(s)
        counts[key] = n + 1
    return selected


@contextmanager
def ro_db(path):
    c = sqlite3.connect(Path(path).resolve().as_uri() + "?mode=ro", uri=True, timeout=1)
    try:
        c.row_factory = sqlite3.Row
        c.execute("PRAGMA query_only=ON")
        yield c
    finally:
        c.close()


def codex_transcript(s, rows, source, now):
    pending = {}
    lifecycle, last_event, saw_start = None, None, False
    for row in rows:
        p = row.get("payload", {})
        if not isinstance(p, dict):
            continue
        at = row.get("timestamp")
        typ = p.get("type")
        if row.get("type") == "event_msg":
            if typ in ("task_started", "turn_started"):
                lifecycle, last_event, saw_start = "inProgress", at, True
                pending.clear()
            elif typ in ("task_complete", "turn_complete", "turn_aborted"):
                lifecycle, last_event = ("interrupted" if typ == "turn_aborted" else "completed"), at
                pending.clear()
            elif typ == "user_message":
                pending.clear()
                message(s, "user", at, p.get("message", ""))
            elif typ == "agent_message":
                message(s, "assistant", at, p.get("message", ""))
        elif row.get("type") == "response_item":
            if typ in ("function_call", "custom_tool_call"):
                q = question_call(p.get("name", ""), p.get("call_id"), p.get("arguments", p.get("input")), at)
                if q:
                    pending[q["call_id"]] = q
            elif typ in ("function_call_output", "custom_tool_call_output"):
                q = pending.get(p.get("call_id"))
                if q and q["blocking"]:
                    pending.pop(p.get("call_id"), None)
            elif typ == "message" and p.get("role") in ("user", "assistant"):
                message(s, p["role"], at, p.get("content"))
        # Any appended event is recency evidence, not proof of process ownership.
        if stamp(at):
            s["_latest_event"] = at
    return pending, lifecycle, last_event, saw_start


def collect_codex(home, roots, limit, now):
    records, warnings = {}, []
    dbs = sorted(home.glob("state_*.sqlite")) + sorted((home / "sqlite").glob("state_*.sqlite"))
    for db in dbs:
        try:
            with ro_db(db) as c:
                cols = {r[1] for r in c.execute("PRAGMA table_info(threads)")}
                wanted = [x for x in ("id", "cwd", "title", "name", "updated_at", "updated_at_ms", "archived", "source", "originator", "rollout_path", "agent_path") if x in cols]
                if not {"id", "cwd", "rollout_path"}.issubset(cols):
                    warnings.append(f"Unsupported threads schema: {db}")
                    continue
                for row in c.execute("SELECT " + ",".join(wanted) + " FROM threads"):
                    d = dict(row)
                    if "subagent" in (d.get("source") or "") or d.get("agent_path") not in (None, "", "/root") or not scoped(d.get("cwd"), roots):
                        continue
                    age = stamp(d.get("updated_at_ms") or d.get("updated_at")) or 0
                    prev = records.get(d["id"])
                    if not prev or age > prev["_sort"]:
                        records[d["id"]] = dict(d, _sort=age, _db=str(db))
        except (OSError, sqlite3.Error) as e:
            warnings.append(f"Codex metadata unavailable at {db}: {type(e).__name__}")
    # Filesystem fallback also supplies the legacy desktop originator absent from old DB rows.
    if not records and not dbs:
        files = sorted((home / "sessions").glob("**/rollout-*.jsonl"), key=lambda p: p.stat().st_mtime, reverse=True)
        if len(files) > 2000:
            warnings.append("Codex fallback discovery limited to 2000 recent rollouts")
        for path in files[:2000]:
            try:
                with path.open() as f:
                    d = json.loads(f.readline()).get("payload", {})
                if not scoped(d.get("cwd"), roots) or not isinstance(d.get("source"), str) or d.get("source") == "cli":
                    continue
                records[d["id"]] = dict(d, title=d["id"], rollout_path=str(path), _sort=path.stat().st_mtime, _db=str(path))
            except (OSError, ValueError, KeyError):
                continue
    candidates = []
    for d in records.values():
        path = Path(d["rollout_path"])
        origin = d.get("originator")
        if not origin:
            try:
                with path.open() as f:
                    origin = json.loads(f.readline()).get("payload", {}).get("originator")
            except (OSError, ValueError):
                pass
        client = "desktop" if origin and "desktop" in origin.lower() else "unknown"
        if d.get("source") in ("cli", "exec") and client != "desktop":
            continue
        s = new_session("codex", d["id"], d["cwd"], d.get("name") or d.get("title"), d["_db"], d["_sort"], client)
        s["_rollout"], s["_db"] = str(path), d["_db"]
        candidates.append(s)
    selected = select_recent(candidates, limit)
    for s in selected:
        path = Path(s.pop("_rollout"))
        s["transcript_path"] = str(path)
        pending, lifecycle, last_event, saw_start = {}, None, None, False
        truncated = True
        try:
            rows, truncated, malformed = tail_rows(path)
            add_tail_warnings(s, path, truncated, malformed)
            pending, lifecycle, last_event, saw_start = codex_transcript(s, rows, path, now)
        except OSError:
            s["warnings"].append("Rollout is unavailable")
        db = Path(s.pop("_db"))
        history = db.parent / "thread_history_1.sqlite"
        if not history.exists():
            history = home / "thread_history_1.sqlite"
        if history.exists():
            try:
                with ro_db(history) as c:
                    turn = c.execute("SELECT status,started_at,completed_at FROM thread_turns WHERE thread_id=? ORDER BY rollout_ordinal DESC LIMIT 1", (s["id"],)).fetchone()
                    if turn:
                        at = turn["completed_at"] or turn["started_at"]
                        # Projection can lag rollout writes. Never overwrite newer terminal evidence.
                        if not last_event or (stamp(at) or 0) > (stamp(last_event) or 0):
                            lifecycle, last_event = turn["status"], at
            except sqlite3.Error:
                s["warnings"].append("Thread history schema unavailable; using rollout")
        s["last_turn_status"] = lifecycle
        latest = s.pop("_latest_event", last_event)
        terminal = lifecycle in ("completed", "failed", "interrupted")
        recent = bool(stamp(latest) and -5 <= now - stamp(latest) < FRESH_SECONDS)
        if terminal:
            s["activity"] = finding("idle", path, last_event, "inferred", "Last persisted turn ended; does not prove the thread is currently loaded")
            pending.clear()
        elif lifecycle == "inProgress" and recent:
            s["activity"] = finding("running", path, latest, "inferred", "Open turn with recent events; live Desktop ownership not verified")
        else:
            s["activity"] = finding(source=path, at=latest, reason="Persisted open turn or incomplete history; no recent live evidence")
        finish_question(s, pending, path, latest, live=recent and lifecycle == "inProgress", terminal=terminal, complete_tail=not truncated or saw_start)
        s["process"] = finding(reason="Codex app-server shares many threads; process existence cannot identify a live turn")
    return selected, warnings, len(candidates)


def claude_transcript(s, rows):
    pending, terminal, latest = {}, False, None
    for row in rows:
        typ, at = row.get("type"), row.get("timestamp")
        if at:
            latest = at
        msg = row.get("message", {})
        if not isinstance(msg, dict):
            continue
        content = msg.get("content", [])
        if typ in ("user", "assistant"):
            message(s, typ, at, content)
        for b in content if isinstance(content, list) else []:
            if not isinstance(b, dict):
                continue
            if b.get("type") == "tool_use":
                terminal = False
                q = question_call(b.get("name", ""), b.get("id"), b.get("input"), at)
                if q:
                    pending[q["call_id"]] = q
            elif b.get("type") == "tool_result":
                pending.pop(b.get("tool_use_id"), None)
        if typ == "assistant" and msg.get("stop_reason") == "end_turn":
            terminal = True
        elif typ == "user" and clean_text(content).strip():
            terminal = False
            pending.clear()
    return pending, terminal, latest


def collect_claude(home, app, roots, limit, now, probe=process_probe):
    candidates, warnings, registry = [], [], {}
    for p in (home / "sessions").glob("*.json"):
        try:
            d = read_json(p)
            if d.get("entrypoint") == "claude-desktop":
                registry[d.get("sessionId")] = (d, p)
        except (OSError, ValueError, AttributeError):
            warnings.append(f"Unreadable Claude registry: {p.name}")
    for p in (app / "claude-code-sessions").glob("*/*/local_*.json"):
        try:
            d = read_json(p)
            if not scoped(d.get("cwd"), roots):
                continue
            sid = d.get("cliSessionId") or d.get("sessionId")
            s = new_session("claude", sid, d["cwd"], d.get("title"), p, d.get("lastActivityAt"), "desktop")
            s["app_session_id"] = d.get("sessionId")
            candidates.append(s)
        except (OSError, ValueError, AttributeError):
            warnings.append(f"Unreadable Claude Desktop metadata: {p.name}")
    known = {s["id"] for s in candidates}
    for sid, (d, p) in registry.items():
        if sid not in known and scoped(d.get("cwd"), roots):
            s = new_session("claude", sid, d["cwd"], d.get("name"), p, d.get("updatedAt"), "desktop")
            s["app_session_id"] = d.get("hostSessionId")
            candidates.append(s)
    probes = {sid: probe(d, "claude") for sid, (d, _) in registry.items() if scoped(d.get("cwd"), roots)}
    selected = select_recent(candidates, limit, {sid for sid, p in probes.items() if p["value"] == "alive"})
    for s in selected:
        files = list((home / "projects").glob("*/" + s["id"] + ".jsonl"))
        pending, terminal, latest, truncated = {}, False, None, True
        if files:
            path = max(files, key=lambda p: p.stat().st_mtime)
            try:
                rows, truncated, malformed = tail_rows(path)
                add_tail_warnings(s, path, truncated, malformed)
                pending, terminal, latest = claude_transcript(s, rows)
            except OSError:
                s["warnings"].append("Transcript unavailable")
        else:
            path = Path(s["source"])
            s["warnings"].append("No linked local Claude Code transcript")
        s["transcript_path"] = str(path) if files else None
        native_wait = False
        if s["id"] in registry:
            d, rp = registry[s["id"]]
            s["process"] = probes.get(s["id"], finding())
            s["native_status"] = d.get("status")
            if s["process"]["value"] == "alive":
                status = {"busy": "running", "waiting": "waiting", "idle": "idle"}.get(d.get("status"), "unknown")
                native_wait = status == "waiting"
                s["activity"] = finding(status, rp, d.get("statusUpdatedAt"), "observed", "Native status with PID and process-start verification")
            elif s["process"]["value"] == "stopped":
                s["activity"] = finding("stopped", rp, now, "observed", "Registered CLI process ended; persisted state may be stale")
        elif terminal:
            s["activity"] = finding("idle", path, latest, "inferred", "Latest transcript ends a turn; current process unknown")
        s["last_turn_status"] = "completed" if terminal else None
        finish_question(s, pending, path, latest, live=native_wait, terminal=terminal, complete_tail=not truncated)
        if native_wait and not pending:
            s["question"] = finding(source=path, at=latest, reason="Native waiting status does not distinguish question from approval")
    return selected, warnings, len(candidates)


def grok_updates(s, rows):
    pending, terminal, latest, turn_end = {}, False, None, None
    stream_role, stream_text = None, ""
    for row in rows:
        params = row.get("params", {})
        u = params.get("update", {})
        if not isinstance(u, dict):
            continue
        at = params.get("_meta", {}).get("agentTimestampMs") or row.get("timestamp")
        if at:
            latest = at
        kind = u.get("sessionUpdate")
        if kind in ("user_message_chunk", "agent_message_chunk"):
            role = "user" if kind == "user_message_chunk" else "assistant"
            if role != stream_role:
                stream_role, stream_text = role, ""
            text = clean_text([u.get("content", {})])
            stream_text = (stream_text + text)[-16000:]
            if role == "user":
                terminal = False
                pending.clear()
            message(s, role, at, stream_text)
        elif kind == "tool_call":
            stream_role, stream_text = None, ""
            q = question_call(u.get("title", ""), u.get("toolCallId"), u.get("rawInput"), at)
            if q:
                pending[q["call_id"]] = q
        elif kind == "tool_call_update" and u.get("status") in ("completed", "failed", "cancelled"):
            pending.pop(u.get("toolCallId"), None)
        elif kind == "turn_completed":
            terminal, turn_end = True, at
            pending.clear()
    return pending, terminal, latest, turn_end


def collect_grok(home, roots, limit, now, probe=process_probe):
    candidates, warnings, registry = [], [], {}
    active = home / "active_sessions.json"
    if active.exists():
        try:
            for d in read_json(active):
                registry[d["session_id"]] = d
        except (OSError, ValueError, KeyError, TypeError):
            warnings.append("Grok active-session registry unavailable")
    for p in (home / "sessions").glob("*/*/summary.json"):
        try:
            d = read_json(p)
            info = d.get("info", {})
            if d.get("parent_session_id") or str(d.get("session_kind", "")).startswith("subagent") or not scoped(info.get("cwd"), roots):
                continue
            s = new_session("grok", info["id"], info["cwd"], d.get("generated_title") or d.get("session_summary"), p, d.get("last_active_at") or d.get("updated_at"), "headless" if d.get("session_kind") == "headless" else "cli")
            s["session_kind"] = d.get("session_kind", "unknown")
            candidates.append(s)
        except (OSError, ValueError, KeyError, AttributeError):
            warnings.append(f"Unreadable Grok summary: {p.parent.name}")
    probes = {sid: probe(d, "grok") for sid, d in registry.items() if scoped(d.get("cwd"), roots)}
    selected = select_recent(candidates, limit, {sid for sid, p in probes.items() if p["value"] == "alive"})
    for s in selected:
        folder = Path(s["source"]).parent
        path = folder / "updates.jsonl"
        s["transcript_path"] = str(folder / "chat_history.jsonl")
        pending, terminal, latest, turn_end, truncated = {}, False, None, None, True
        try:
            rows, truncated, malformed = tail_rows(path)
            add_tail_warnings(s, path, truncated, malformed)
            pending, terminal, latest, turn_end = grok_updates(s, rows)
        except OSError:
            s["warnings"].append("Grok updates unavailable")
        s["last_message_update_at"] = s["last_message"]["at"]
        # Final chat messages avoid reporting the last streaming token as a summary.
        try:
            chat, _, malformed = tail_rows(folder / "chat_history.jsonl")
            for row in chat:
                if row.get("type") in ("user", "assistant") and clean_text(row.get("content")).strip():
                    text = clean_text(row.get("content"))
                    previous = s["last_message"]
                    # Keep the real streamed-message timestamp only when the content agrees.
                    at = previous["at"] if previous["role"] == row["type"] and previous["excerpt"] and (previous["excerpt"][:200] in text or previous["excerpt"][-160:] in text) else None
                    s["_chat_message"] = dict(at=at, role=row["type"], excerpt=text[:400])
            if "_chat_message" in s:
                s["last_message"] = s.pop("_chat_message")
            # Chat rows lack timestamps; do not assign a later tool event's time to the message.
        except OSError:
            pass
        phase, phase_at, outcome = None, None, None
        try:
            events, ev_truncated, malformed = tail_rows(folder / "events.jsonl")
            add_tail_warnings(s, folder / "events.jsonl", ev_truncated, malformed)
            for row in events:
                if row.get("type") == "turn_started":
                    phase, phase_at, outcome = "running", row.get("ts"), None
                elif row.get("type") == "phase_changed":
                    phase, phase_at = row.get("phase"), row.get("ts")
                elif row.get("type") == "turn_ended":
                    phase, phase_at, outcome = "ended", row.get("ts"), row.get("outcome")
            if phase == "ended" and (not latest or (stamp(phase_at) or 0) >= (stamp(latest) or 0)):
                terminal, turn_end = True, phase_at
                pending.clear()
        except OSError:
            pass
        if phase != "ended" and stamp(phase_at) and stamp(phase_at) > (stamp(turn_end) or 0):
            terminal = False
        if stamp(phase_at) and stamp(phase_at) > (stamp(latest) or 0):
            latest = phase_at
        live = s["id"] in probes and probes[s["id"]]["value"] == "alive"
        if s["id"] in probes:
            s["process"] = probes[s["id"]]
        s["last_turn_status"] = outcome or ("completed" if terminal else None)
        if terminal:
            s["activity"] = finding("idle" if live else "stopped" if s["process"]["value"] == "stopped" else "idle", path, turn_end, "inferred", "Last recorded turn ended" + ("; process is alive" if live else "; current process unverified"))
        elif live:
            fresh = stamp(latest) and -5 <= now - stamp(latest) < FRESH_SECONDS
            s["activity"] = finding("running" if fresh else "unknown", path, latest, "inferred", "Live process with recent updates" if fresh else "Live process but no recent turn evidence")
        elif s["process"]["value"] == "stopped":
            s["activity"] = finding("stopped", active, now, "observed", "Registered process no longer exists")
        if live and phase == "permission_prompt" and stamp(phase_at) and now - stamp(phase_at) < FRESH_SECONDS:
            s["permission"] = finding("unknown", folder / "events.jsonl", phase_at, "inferred", "Permission phase can auto-resolve; inspect current UI before claiming a prompt")
        finish_question(s, pending, path, latest, live=live, terminal=terminal, complete_tail=not truncated)
        s["last_event_at"] = iso(latest)
    return selected, warnings, len(candidates)


def collect(home, roots, limit=3, now=None):
    now = now or time.time()
    roots = {str(Path(p).resolve()) for p in roots}
    report = dict(schema_version=1, observed_at=iso(now), sessions=[], sources=[], limitations=[
        "No hooks, network requests, agent invocations, or UI inspection.",
        "Activity and questions are separate. Unknown is not idle or no.",
        "Local Codex threads, Claude Desktop Code chats, and Grok CLI only; cloud-only chats and Claude Chat/Cowork are not covered.",
        f"Latest {limit or "all"} sessions per provider/folder plus registered live Claude/Grok sessions; subagents excluded."])
    providers = [
        ("codex", Path(os.environ.get("CODEX_HOME", home / ".codex")), lambda p: collect_codex(p, roots, limit, now)),
        ("claude", Path(os.environ.get("CLAUDE_CONFIG_DIR", home / ".claude")), lambda p: collect_claude(p, home / "Library/Application Support/Claude", roots, limit, now)),
        ("grok", Path(os.environ.get("GROK_HOME", home / ".grok")), lambda p: collect_grok(p, roots, limit, now)),
    ]
    for engine, path, fn in providers:
        try:
            sessions, warnings, count = fn(path)
            report["sessions"].extend(sessions)
            report["sources"].append(dict(engine=engine, path=str(path), status=("partial" if warnings else "available") if path.exists() else "missing", discovered=count, returned=len(sessions), warnings=warnings))
        except (OSError, ValueError, sqlite3.Error, TypeError, KeyError, AttributeError) as e:
            report["sources"].append(dict(engine=engine, path=str(path), status="unknown", discovered=None, returned=0, warnings=[f"Collector failed: {type(e).__name__}"]))
    for s in report["sessions"]:
        s["unknown_fields"] = [k for k in ("activity", "question", "permission", "process", "spec_association") if s[k]["value"] == "unknown"]
        if not s["last_message"]["at"]:
            s["unknown_fields"].append("last_message.at")
        if s["client"] == "unknown":
            s["unknown_fields"].append("client")
    return report


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--home", type=Path, default=Path.home())
    p.add_argument("--limit", type=int, default=3)
    p.add_argument("roots", nargs="*")
    args = p.parse_args()
    if args.limit < 0:
        p.error("--limit must be nonnegative")
    print(json.dumps(collect(args.home, args.roots, args.limit), indent=2))


if __name__ == "__main__":
    main()
