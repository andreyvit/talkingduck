# Collect and enrich agent work

Instructions for the agent already handling the user's request. Do not launch
another agent or model from the collector. This file is not an installed skill.

## Run the local baseline

From TalkingDuck:

```sh
go run ./scripts/inventory -json
```

Default scope: Prairie `backend`, `back2` through `back10`. Supply explicit
checkout paths for a different scope; `-agent-limit 5` limits recent sessions; the default scans all candidates.
Use `-no-agents` for Git/spec-only output. Standalone session collection:

```sh
go run ./scripts/inventory -agents-only /absolute/checkout/path
```

Read `agent_sources`, their warnings, and `agent_error` before interpreting missing
sessions. Each spec has `sessions`; each repository also has `unassigned_sessions`. Each session has `activity`, `question`,
`permission`, `process`, `spec_association`, timestamps, evidence, and
`unknown_fields`. Unknown is not idle, no, or an empty answer. Native formats can
change; return unknown rather than repairing the user's app data.

JSON output contains private titles, questions, and bounded message excerpts.
Keep saved observations in `.tasks/` or another local ignored destination.
Do not upload or commit them. Inspect only session files needed for the request.

## Enrich cheaply first

1. Report explicit current questions first. Quote `pending_questions[].prompts`,
   preserving session, engine, full folder, and source time.
2. Read the latest assistant text when question state is unknown because of prose.
   Decide whether it actually asks for an answer or merely mentions a question.
   A question mark alone is not enough. State your inference and source.
3. For Codex, use available app tools before UI: `list_threads` (maximum limit 50),
   match exact thread ID and host, then `read_thread` for recent status/items.
   `wait_threads` with timeout zero can provide a current snapshot. Do not send a
   message, resume a turn, create a thread, or start another app-server to observe
   the existing one. `notLoaded` does not mean the task is done.
4. Treat a session's folder as evidence of location, not spec ownership. Read a
   recent explicit spec reference if association matters. Preserve ambiguity when
   several specs are named without unique ticket corroboration. Never assign the newest spec by default.
5. Native process state already enriches Claude and Grok. If it disagrees with
   newer transcript evidence, preserve both observations and mark the disputed
   field unknown. Don't use cmux's badge to break the tie.

## UI fallback for unresolved fields

Use computer use only when the missing field matters to the user's request.
Read the app's accessibility tree first. Screenshots are for missing visual detail.
Use the available computer-use tool; do not add screen automation to the collector.

- **Codex Desktop:** locate the exact thread and host. Inspect Stop/working state,
  question cards, and approval controls. The app may be named ChatGPT. Prefer its
  read-only tools when available.
- **Claude Desktop:** use Code mode and the exact title/`app_session_id`. Confirm
  an actual question dialog with response options, or a permission prompt.
  “Awaiting input” in the sidebar alone is not proof; it can change merely when
  viewed. A Stop button can coexist with a question wait.
- **Grok Build in cmux:** identify the workspace/surface by folder and session.
  Read the terminal through cmux's read-only surface API/CLI if available, otherwise
  inspect its visible terminal. Look for the current question/permission widget,
  not an earlier prompt in scrollback. Never type a probe into an active session.

Do not answer, dismiss, cancel, resume, approve, or send anything during collection.
Avoid moving the user away from their current view when a read-only query suffices.
Opening a chat may change its read state; restore the prior view when practical.

Keep enrichments separate from the baseline. An enrichment record should contain:

```json
{
  "engine": "claude",
  "session_id": "session-id-from-baseline",
  "field": "question",
  "value": "yes",
  "source": "Claude Desktop question dialog",
  "observed_at": "2026-10-02T12:00:00Z",
  "confidence": "observed",
  "evidence": "Exact visible question text"
}
```

This is a caller-side reporting convention, not an input format consumed by the
script. Do not erase conflicting baseline evidence. Rerun the baseline if events
changed while inspecting. If the field remains unresolved, leave it unknown and
say which source could resolve it. No hook installation is required or implied.
