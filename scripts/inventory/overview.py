"""Correlate local sessions to selected specs; render a compact worktree/task table."""
import argparse
import datetime as dt
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import unicodedata


def git(root, *args):
    p = subprocess.run(["git", "-C", root, *args], capture_output=True, timeout=15,
                       env=dict(os.environ, GIT_OPTIONAL_LOCKS="0"))
    if p.returncode:
        raise ValueError("Git observation unavailable: " + " ".join(args[:2]))
    return p.stdout.decode("utf-8", "replace")


def timestamp(value):
    try:
        return dt.datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp() if value else 0
    except (TypeError, ValueError, AttributeError):
        return 0


def section(text, heading):
    m = re.search(r"^## " + re.escape(heading) + r"\s*\n(.*?)(?=^## |\Z)", text, re.M | re.S | re.I)
    return m[1] if m else ""


def task_state(spec, text):
    subtasks = re.findall(r"^(?:\d+[.)] |[-*] |### )\[([ xX>])\]", section(text, "Subtasks"), re.M)
    done = sum(x.lower() == "x" for x in subtasks)
    pending = section(text, "Pending suggestions")
    # Ignore template instructions, resolved checkboxes, and nested explanation lists.
    pending = re.sub(r"<[^>]*>", "", pending, flags=re.S)
    items = re.findall(r"^(?:[-*] |\d+[.)] )(.*\S)", pending, re.M)
    suggestions = sum(not re.match(r"\[[xX]\]|~~|(?:None|No pending)\b", x, re.I) for x in items)
    residual = [x for x in pending.splitlines() if x.strip() and not x.startswith(("#", " ", "-", "*")) and not re.match(r"(?:\d+[.)]|None\.?$|No (?:pending|open|suggestions))", x, re.I)]
    if not items and residual:
        suggestions = None  # Prose is not a reliable count.
    status = spec.get("status", "unknown").lower()
    log = re.sub(r"<[^>]*>", "", section(text, "Log"), flags=re.S)
    events = re.findall(r"^(?:[-*] |\d+[.)] )(.+)$", log, re.M)
    revision_events = list(dict.fromkeys(e for e in events if re.search(
        r"\buser revision\b|\b(?:user|human) (?:requested|requests|asked for) (?:a |another |major |new )*revision\b|\b(?:major|task) (?:revision|iteration) (?:requested|started)\b", e, re.I)))
    substantive = any(re.sub(r"<[^>]*>|^#+[^\n]*$", "", section(text, h), flags=re.M | re.S).strip()
                      for h in ("High-level plan", "Low-level plan", "Product Spec", "Spec"))
    round_number = 1 if subtasks or substantive or any(not re.search(r"^(?:created|create|initialized) (?:task|spec)", e, re.I) for e in events) else 0
    round_number += len(revision_events)
    explicit_rounds = [int(m[1]) for e in events for m in [re.search(r"\b(?:task round|task iteration|major iteration)\s*[:#]?\s*(\d+)\b", e, re.I)] if m]
    if explicit_rounds:
        round_number = max(round_number, explicit_rounds[-1])
    all_done = bool(subtasks) and done == len(subtasks)
    completed = status in ("done", "complete", "completed")
    if completed or all_done:
        stage = "FIN"
    elif "final" in status and "review" in status:
        stage = "FIN"
    elif "subtask" in status or "execut" in status or "implement" in status:
        stage = "EX"
    elif "low-level" in status or "low level" in status:
        stage = "HL" if re.search(r"waiting.*before low.level", status) else "LL"
    elif "high-level" in status or "high level" in status:
        stage = "HL"
    elif status == "planning":
        stage = "LL" if subtasks else "HL"
    elif "rebas" in status:
        stage = "PUSH" if "push" in status else "FIN" if "closeout" in status else "REBA"
    else:
        stage = "--"
    latest_action = None
    squashed_this_round = False
    operation_context = None
    # Log is chronological (newest last). Review-loop numbers are not task rounds.
    for event in events:
        low = event.lower()
        if event in revision_events or re.search(r"\b(?:task round|task iteration|major iteration)\s*[:#]?\s*\d+\b", event, re.I):
            squashed_this_round = False
            operation_context = None
            latest_action = dict(stage="HL" if re.search(r"replan|high.level|planning", low) else "EX" if subtasks and not all_done else stage if stage in ("HL", "LL") else "FIN", entry=event)
            continue
        action = None
        if re.search(r"\b(?:zoo[- ]closeout|closeout (?:started|running|complete)|(?:started|running|completed) closeout)\b", low):
            operation_context = "FIN"
            action = "FIN"
        elif re.search(r"\b(?:zoo-push|push (?:workflow|started|running)|(?:started|running) push)\b", low):
            operation_context = "PUSH"
            action = "PUSH"
        rebase = bool(re.search(r"\b(?:rebased|rebasing|rebase (?:started|running|finished|completed?|done)|(?:started|running|completed|finished) (?:zoo-)?rebase)\b", low))
        if rebase:
            if re.search(r"(?:part of|during|for) (?:zoo[- ]|the )?push|zoo-push", low):
                operation_context = "PUSH"
            elif "closeout" in low:
                operation_context = "FIN"
            action = operation_context or "REBA"
        elif re.search(r"\b(?:squashed|squeezed)\b|\b(?:squash|squeeze)(?: ran| complete| done| succeeded)\b|\b(?:ran|completed) (?:zoo-)?(?:squash|squeeze)\b", low) and not re.search(r"\b(?:before|pre-squash|pre-squeeze|not|never|will|plan to|should)\b", low):
            action = "SQ"
        elif re.search(r"\b(?:ready (?:to|for) push|awaiting push|push approved)\b", low):
            action = "PUSH"
        elif re.search(r"\b(?:final|uber)[ -]review\b", low):
            action = "FIN" if all_done or completed else None
        elif re.search(r"\b(?:subtask \d+ (?:started|complete|done)|(?:started|completed|executing) subtask \d+)\b", low):
            action = "FIN" if all_done else "EX"
        elif re.search(r"\blow-level (?:plan|spec review)\b", low) and not re.search(r"low.level.*(?:placeholder|not started|deferred)|before low.level", low):
            action = "LL"
        elif re.search(r"\bhigh-level (?:plan|spec review)\b", low):
            action = "HL"
        if action and not rebase and action not in ("PUSH", "FIN"):
            operation_context = None
        if action == "SQ" and all_done:
            squashed_this_round = True
        if action:
            latest_action = dict(stage=action, entry=event)
    if latest_action:
        if latest_action["stage"] in ("SQ", "PUSH"):
            if all_done or latest_action["stage"] == "PUSH":
                stage = latest_action["stage"]
        elif not (all_done or completed) or latest_action["stage"] in ("FIN", "REBA"):
            stage = latest_action["stage"]
    ready = completed or bool(re.search(r"awaiting|waiting for|ready|approved$|investigation complete", status)) or stage in ("SQ", "PUSH")
    return dict(stage=stage, round=round_number, round_evidence=revision_events,
                round_confidence="inferred", squashed_this_round=squashed_this_round, latest_logged_action=latest_action,
                subtasks_done=done, subtasks_total=len(subtasks), suggestions=suggestions,
                progress="✓" if ready or (latest_action and re.search(r"complete|finished|passed|ready", latest_action["entry"], re.I)) else "?", source=spec["path"],
                reason="Inferred from chronological spec Log, status, and top-level sections; missing history can undercount rounds")


def info_file(path):
    parts = Path(path).parts
    return path.lower().endswith(".md") and any(p in (".spec", ".proposal", ".proposals", ".tasks") for p in parts)


def git_state(repo):
    tracked, untracked, info = set(), set(), set()
    for c in repo.get("local_changes", []):
        p = c["path"]
        if c["status"] == "rename/copy source":
            continue
        if c["status"] != "??":
            tracked.add(p)
        if info_file(p):
            info.add(p)
        elif c["status"] == "??":
            untracked.add(p)
    warnings = []
    try:
        behind, ahead = map(int, git(repo["path"], "rev-list", "--left-right", "--count", "@{upstream}...HEAD").split())
    except (OSError, ValueError, subprocess.TimeoutExpired):
        behind = ahead = None
    changes = tracked | untracked | info
    cleanliness = "CLEAN" if all(info_file(p) for p in changes) else "MARKDN" if all(p.lower().endswith(".md") for p in changes) else "DIRTY"
    return dict(tracked=len(tracked), untracked=len(untracked), informational=len(info),
                ahead=ahead, behind=behind, cleanliness=cleanliness, warnings=warnings,
                scope="Current staged, unstaged, and untracked paths only; remote counts use local upstream ref")


def text_content(value):
    if isinstance(value, str):
        return value
    if isinstance(value, list):
        return "\n".join(x.get("text", "") for x in value if isinstance(x, dict) and isinstance(x.get("text"), str))
    return ""


def transcript_evidence(session):
    """Only conversation text: never tool output, injected instructions, or reasoning."""
    path = session.get("transcript_path")
    if not path:
        return [], False
    rows = []
    try:
        with open(path, "rb") as f:
            size = os.fstat(f.fileno()).st_size
            head = f.read(256 * 1024)
            if size > 4 * 1024 * 1024:
                f.seek(size - 4 * 1024 * 1024)
                tail = f.read().partition(b"\n")[2]
                data = head.rpartition(b"\n")[0] + b"\n" + tail
            else:
                f.seek(0)
                data = f.read()
        for line in data.splitlines():
            try:
                row = json.loads(line)
            except (ValueError, UnicodeError):
                continue
            typ = row.get("type")
            if session["engine"] == "codex":
                p = row.get("payload", {})
                if typ == "event_msg" and p.get("type") in ("user_message", "agent_message"):
                    role = "user" if p["type"] == "user_message" else "assistant"
                    text = text_content(p.get("message"))
                elif typ == "response_item" and p.get("type") == "message" and p.get("role") in ("user", "assistant"):
                    role, text = p["role"], text_content(p.get("content"))
                else:
                    continue
            elif typ in ("user", "assistant"):
                role = typ
                text = text_content(row.get("message", row).get("content"))
            else:
                continue
            text = re.sub(r"<(rules|user_info|environment_context|system-reminder)>.*?</\1>", "", text, flags=re.S)
            if text.startswith(("# AGENTS.md instructions", "<environment_context>")):
                continue
            if text.strip():
                rows.append((role, text, row.get("timestamp")))
        return rows, size > 4 * 1024 * 1024
    except OSError:
        return [], False


def skill_activity(rows):
    current = None
    for role, text, at in rows:
        low = text.lower()
        # Short explicit requests avoid treating pasted skill instructions as execution.
        invoked = re.search(r"(?:\$|/|\brun\s+(?:\[\$?)?|\buse\s+(?:\[\$?)?|\binvoke\s+)(zoo-(?:rebase|push))\b", low)
        if role == "user" and invoked and len(text) < 5000 and not re.search(r"\b(?:do not|don't|never|not yet)\b", low):
            name = invoked[1]
            current = dict(skill=name, stage="PUSH" if name == "zoo-push" else "REBA",
                           state="requested", at=at, evidence=text[:500], confidence="inferred")
            continue
        if not current:
            continue
        if role == "user" and not re.fullmatch(r"\s*(?:continue|proceed|retry|yes|ok|okay|thanks)[.!]?\s*", low):
            current = None
            continue
        if role != "assistant":
            continue
        if re.search(r"rebas", low) and re.search(r"(?:part of|during|within) (?:the )?(?:zoo[- ]?)?closeout", low):
            current["stage"] = "FIN"
        if re.search(r"rebas", low) and re.search(r"(?:part of|during|within) (?:the )?(?:zoo[- ]?)?push", low):
            current["stage"] = "PUSH"
        completed = re.search(r"(?:^rebased\b|\bi(?: have|\'ve)? rebased\b|\brebase (?:is |has )?(?:complete[ds]?|finished|done)|(?:completed|finished) (?:the )?(?:zoo-)?rebase)\b", low) if current["skill"] == "zoo-rebase" else re.search(r"\b(?:pushed|push (?:complete[ds]?|finished|done))\b", low)
        if completed and not re.search(r"\b(?:will|not|never|before|plan to)\b", low):
            current.update(state="completed", at=at, evidence=text[:500])
    return current


def associate(session, specs):
    rows, bounded = transcript_evidence(session)
    session["zoo_skill_activity"] = skill_activity(rows)
    prompts = [text for role, text, _ in rows if role == "user"]
    first = "\n".join(prompts[:3])
    review = bool(re.search(r"findings only|do not (?:modify|edit) files|do not implement|read.only review", first, re.I) and
                  re.search(r"review|zoo-ensure-safe-deploy", first, re.I))
    if session.get("parent_id"):
        role, reason = "secondary", "Explicit parent session"
    elif session.get("client") == "headless" and review:
        role, reason = "secondary", "Headless session with review-only prompt"
    elif session.get("client") == "headless":
        role, reason = "unknown", "Headless does not establish whether this is a primary task"
    elif session.get("client") in ("desktop", "cli"):
        role, reason = "primary", "Top-level interactive session; inferred from client metadata"
    else:
        role, reason = "unknown", "Client role unavailable"
    session["task_role"] = dict(value=role, reason=reason, confidence="inferred")
    if bounded:
        session.setdefault("warnings", []).append("Association uses transcript head and tail; middle omitted")
    # Explicit path references win. Several references are ambiguous, never resolved by recency.
    hits = {}
    for spec in specs:
        path = spec["path"]
        base = Path(path).name
        for speaker, text, at in rows:
            if re.search(r"(?<![\w-])" + re.escape(base) + r"(?![\w.-])", text):
                hits[path] = dict(kind="explicit spec reference", at=at, speaker=speaker)
    if not hits:
        title_tickets = set(re.findall(r"\b[A-Z][A-Z0-9]+-\d+\b", session.get("title", "").upper()))
        for spec in specs:
            if title_tickets.intersection(spec.get("tickets", [])):
                hits[spec["path"]] = dict(kind="unique ticket in session title", at=session.get("updated_at"))
    if len(hits) > 1:
        title_tickets = set(re.findall(r"\b[A-Z][A-Z0-9]+-\d+\b", session.get("title", "").upper()))
        supported = [spec["path"] for spec in specs if spec["path"] in hits and title_tickets.intersection(spec.get("tickets", []))]
        if len(supported) == 1:
            hits = {supported[0]: dict(hits[supported[0]], corroboration="Unique ticket in session title")}
    if len(hits) == 1:
        path, evidence = next(iter(hits.items()))
        session["spec_association"] = dict(value=path, confidence="inferred", source=session.get("transcript_path") or session["source"], **evidence)
        session["unknown_fields"] = [x for x in session.get("unknown_fields", []) if x != "spec_association"]
        return path
    session["spec_association"] = dict(value="ambiguous" if hits else "unknown", candidates=list(hits),
                                       reason="Multiple spec references" if hits else "No explicit spec or unique title ticket match")
    return None


def enrich(report):
    report["schema_version"] = 2
    for repo in report["repositories"]:
        repo["git"] = git_state(repo) if not repo.get("error") else dict(error=repo["error"])
        specs = repo.get("specs", [])
        for spec in specs:
            try:
                text = (Path(repo["path"]) / spec["path"]).read_text()
            except OSError:
                text = ""
            spec["workflow"] = task_state(spec, text)
            spec["sessions"] = []
            spec["primary_session"] = None
            # Existing H1/title is preserved; a filename fallback becomes its useful slug.
            spec["display_title"] = re.sub(r"^\d{8}-(?:(?:dev-)?\d+-)*", "", spec["title"], flags=re.I)
        unassigned = []
        for session in repo.pop("agents", []) or []:
            path = associate(session, specs)
            if path:
                next(s for s in specs if s["path"] == path)["sessions"].append(session)
            else:
                unassigned.append(session)
        repo["unassigned_sessions"] = unassigned
        for spec in specs:
            primaries = sorted((s for s in spec["sessions"] if s["task_role"]["value"] == "primary"),
                               key=lambda s: (timestamp(s.get("updated_at")), s["id"]), reverse=True)
            active = [s for s in primaries if s["activity"]["value"] in ("running", "waiting")]
            spec["primary_conflict"] = len(active) > 1
            if primaries:
                primary = primaries[0]
                spec["primary_session"] = dict(engine=primary["engine"], id=primary["id"])
                workflow = spec["workflow"]
                activity = primary["activity"]["value"]
                skill = primary.get("zoo_skill_activity")
                # Chat timestamps may be missing (Grok). Compare session recency with
                # the spec's activity when known; retain the source in either case.
                if skill and timestamp(primary.get("updated_at")) >= timestamp(spec.get("last_activity")):
                    workflow["stage"] = skill["stage"]
                    workflow["skill_evidence"] = skill
                    workflow["progress"] = "✓" if skill["state"] == "completed" else "?"
                if activity == "running":
                    workflow["progress"] = "▶"
                workflow["interaction"] = "!" if activity == "waiting" or primary["question"]["value"] == "yes" else "" if primary["question"]["value"] == "no" else "?"
                workflow["questions"] = sum(len(q.get("prompts", [])) or 1 for q in primary.get("pending_questions", [])) if primary.get("pending_questions") else 0 if primary["question"]["value"] == "no" else None
            else:
                spec["workflow"].update(interaction="?", questions=None)
    return report


def cell_width(text):
    return sum(0 if unicodedata.combining(c) or c in "\ufe0e\ufe0f" else 2 if unicodedata.east_asian_width(c) in ("W", "F") else 1 for c in text)


def clean(value):
    return " ".join(str(value).split())


def wrap(text, width):
    text = clean(text)
    result, current = [], ""
    for char in text:
        if current and cell_width(current + char) > width:
            result.append(current)
            current = ""
        current += char
    return result + [current]


def render(report, width=120):
    rows = []
    for repo in report["repositories"]:
        g = repo["git"]
        if "error" in g:
            rows.append([Path(repo["path"]).name, "--", "--", "0", "--", "--", "?", "?", "?/?", "ERROR", "⇣? ⇡?", g["error"]])
            continue
        local = f'±{g["tracked"]} ?{g["untracked"]} ℹ︎{g["informational"]}'
        remote = f'⇣{g["behind"] if g["behind"] is not None else "?"} ⇡{g["ahead"] if g["ahead"] is not None else "?"}'
        for spec in repo["specs"] or [None]:
            if not spec:
                rows.append([Path(repo["path"]).name, "--", "--", "0", "--", "0/0", "", "", "0/0", local, remote, g["cleanliness"]])
                continue
            w, primary = spec["workflow"], spec["primary_session"]
            agent = {"codex": "Codex", "claude": "Claude", "grok": "Grok"}.get(primary["engine"], "?") if primary else "?"
            if spec["primary_conflict"]:
                agent += "!"
            q = w.get("questions")
            sug = w["suggestions"]
            rows.append([Path(repo["path"]).name, agent, "\n".join(spec["tickets"]) or "--", str(w.get("round", "?")), w["stage"],
                         f'{w["subtasks_done"]}/{w["subtasks_total"]}', w["progress"], w["interaction"],
                         f'{q if q is not None else "?"}/{sug if sug is not None else "?"}', local, remote, spec["display_title"]])
    # Fixed columns across this table; grow only to prevent data loss.
    widths = [6, 6, 8, 1, 4, 5, 1, 1, 3, 9, 5]
    for row in rows:
        for i, value in enumerate(row[:-1]):
            widths[i] = max(widths[i], *(cell_width(clean(line)) for line in value.split("\n")))
    final_width = max(8, width - sum(widths) - len(widths))
    output = []
    for row in rows:
        cells = [[clean(x) for x in value.split("\n")] for value in row[:-1]] + [wrap(row[-1], final_width)]
        for line in range(max(map(len, cells))):
            values = [cell[line] if line < len(cell) else "" for cell in cells]
            output.append((" ".join(value + " " * (widths[i] - cell_width(value)) for i, value in enumerate(values[:-1])) + " " + values[-1]).rstrip())
    return "\n".join(output)


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--json", action="store_true")
    p.add_argument("--width", type=int, default=0)
    args = p.parse_args()
    report = enrich(json.load(sys.stdin))
    if args.json:
        print(json.dumps(report, indent=2))
    else:
        print(render(report, args.width or shutil.get_terminal_size((120, 24)).columns))
        if report.get("agent_error"):
            print(report["agent_error"], file=sys.stderr)
        for source in report.get("agent_sources", []) or []:
            if source.get("status") != "available":
                print(f'{source["engine"]}: {source.get("status", "unknown")} coverage; see JSON source warnings', file=sys.stderr)
        for repo in report["repositories"]:
            for warning in repo.get("warnings", []) + repo["git"].get("warnings", []):
                print(Path(repo["path"]).name + ": " + warning, file=sys.stderr)


if __name__ == "__main__":
    main()
