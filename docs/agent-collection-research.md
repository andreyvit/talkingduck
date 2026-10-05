# Agent collection research

Research and local validation: 2026-10-02. Goal: useful observations without hooks,
then optional enrichment. Implementation: [agents.go](../inventory/agents.go),
with read-only SQLite access through `modernc.org/sqlite`. Collection, correlation,
and rendering run in Go; no Python runtime or cgo. Go downloads module dependencies
on the first build. The collector makes no model call, network call, or recursive agent run.

## Findings

| Client | Baseline available locally | Strongest current-state evidence | Remaining uncertainty |
| --- | --- | --- | --- |
| Codex Desktop | Thread IDs, display names, folders, rollouts, persisted turns/items | Recent open-turn events; caller's existing Codex app tools can supply live status | Old `inProgress` rows survive; Desktop process is shared; asynchronous questions |
| Claude Desktop Code | Desktop metadata, linked CLI transcripts, native PID registry | Native `busy`/`waiting`/`idle`, checked against process identity; unresolved `AskUserQuestion` | Approval versus other waits; ordinary prose questions; missing/remote transcripts |
| Grok Build CLI | Summaries, active PID registry, ACP updates, event journal, chat history | Live registered process plus recent events; unresolved `ask_user_question` | Long silent work; approval auto-resolution; older formats; headless children without parent metadata |

The current collector covers local Codex threads and **Claude Desktop's Code
mode**, not ordinary Claude Chat, Cowork VM sessions, or cloud-only sessions.
Those are explicit coverage gaps, not an assertion that no such chats exist.
Local app tools can list additional Codex cloud/remote tasks when the caller has
that capability. Machine-local files alone cannot provide their current state.

A recent message is evidence of recent activity, not proof of current execution.
A completed turn is not a completed task. A running process is not a running turn.
Question state is separate from activity: an asynchronous question need not block.

## Local sources inspected

Paths below are relative to the user's home unless stated otherwise. Do not copy
transcripts or metadata into tracked fixtures. Tests use synthetic data.

### Codex Desktop

Installed app inspected: ChatGPT `26.928.31416` (build `12553`), containing Codex.
The app name/path can differ from `Codex.app` used by other monitors.

- `.codex/state_5.sqlite`, `threads`: `id`, `cwd`, `name`, `title`, `originator`,
  `source`, `rollout_path`, `updated_at[_ms]`, `archived`, and agent fields.
  `name` is the displayed name; `title` can still be the original prompt.
- A second `.codex/sqlite/state_5.sqlite` exists locally. Scan both and deduplicate
  by thread ID/updated time. This is compatibility handling, not two live engines.
- `.codex/thread_history_1.sqlite`: `thread_turns` has status/start/end timestamps;
  `thread_items` has typed item JSON and timestamps. The baseline reads latest
  turn state; it does not dump the full database or tool output.
- `.codex/sessions/**/rollout-*.jsonl`: `session_meta`, `event_msg`,
  `response_item`; pair function calls and outputs by `call_id`.
- `originator=codex_work_desktop` identifies sampled Desktop sessions even though
  `source=vscode`. `vscode` alone does **not** prove Desktop. Missing originators
  remain `client=unknown`; known CLI/exec and subagent rows are excluded.

Observed: the history DB contained 82 `inProgress` turns during the initial scan,
including old ones. The current research turn was also there. Treating every open
turn as running would create false positives. The collector reports an open turn
with events under two minutes old, or an unresolved tool call, as **inferred running**.
Quiet tool/subagent waits persist until their result or a terminal event. Old open
turns without either signal remain unknown; a crash without a terminal event can
still leave an unresolved call, so this is not verified live ownership.
Terminal history is **inferred idle**, not proof the app has the thread loaded.

Structured synchronous `request_user_input` calls can be paired with their result.
`request_user_input_async` returns immediately, so an output is not an answer;
its current question state stays unknown without a stronger source. Tools wrapped
inside another tool may not expose their question call in a recognizable record.

SQLite opens use `mode=ro` and `query_only`. Normal SQLite reads include the WAL;
`immutable=1` would risk ignoring live WAL content and is deliberately avoided.
Connections are closed after reading. Missing/version-changed schemas become
source warnings or unknowns.

**Stronger access, in preference order:**

1. The calling agent's existing `list_threads`, `read_thread`, and `wait_threads`
   tools. Live inspection in this research returned `active`, `idle`, and
   `notLoaded`. This requires no new agent and is cheaper than UI inspection.
   Keep host ID; don't confuse a remote/cloud task with a local copy.
2. An adapter to the already-running Desktop app-server. The official
   [app-server protocol](https://learn.chatgpt.com/docs/app-server) provides
   `thread/list`, `thread/read`, `thread/loaded/list`, status notifications, and
   server requests. Starting a **new** app-server reads history but does not give
   it ownership of the Desktop app-server's loaded threads. A protocol method
   existing is not proof we can attach to the existing process.
3. Private desktop coordination IPC. The installed bundle contains
   `.codex/ipc/ipc.sock`, `initialize`, `thread-owner-discovery`, versioned
   `thread-stream-state-changed` broadcasts, and follower operations. These are a
   plausible route to a live snapshot, but not a drop-in public read API. No
   private IPC messages were sent. A future observer must prove it cannot claim
   ownership, start/resume turns, or answer pending requests.
4. Optional passive hooks, then caller-operated UI inspection.

Installed bundle source also names `CODEX_SQLITE_HOME`. The baseline supports
`CODEX_HOME` and the two usual SQLite locations; arbitrary separate SQLite homes
are not yet discovered. The app's shared process and writer-lock file existence
are insufficient thread-liveness tests.

### Claude Desktop Code

Installed Desktop `2.16120.0`; running bundled Claude Code `2.1.284`.

- `Library/Application Support/Claude/claude-code-sessions/<account>/<org>/local_*.json`:
  `sessionId` (Desktop), `cliSessionId`, `cwd`, title, `lastActivityAt`, archive flag.
- `.claude/projects/<encoded-project>/<cliSessionId>.jsonl`: user/assistant
  messages and tool calls/results. Read the content's timestamp, not file mtime.
- `.claude/sessions/<pid>.json`: `sessionId`, `hostSessionId`, `entrypoint`, PID,
  `procStart`, `status`, `statusUpdatedAt`, and a messaging socket path.
  Sampled entrypoint is `claude-desktop`; sampled states include `waiting` and
  `idle`; bundled code writes `busy` as well.

The process registry is native Claude data, not a hook we install. Validate PID,
executable, and process start before using it. `procStart` is UTC locally; `ps`
must use `TZ=UTC` and `LC_ALL=C`. Otherwise the local timezone causes false
identity mismatches. Do not mark a long-lived question stale merely because its
status timestamp is yesterday when the same process still reports it.

Observed two registered live sessions with `status=waiting` and an unresolved
`AskUserQuestion`. The Blueland session's visible dialog matched the parsed
question. No answer, prompt, or approval was submitted during validation.

The UI also initially labeled the finished XSS session “Awaiting input.” Opening
it changed the badge to “Idle”; no question dialog existed. Therefore that sidebar
badge alone is insufficient evidence of an open question. The original chat view
was restored; viewing a chat can still change its read state.

Installed bundle inspection found `pending_permission_requests` and
`pending_user_dialog_requests` in control-protocol initialization, and a native
peer messaging socket. That suggests a richer future adapter, but attaching or
initializing a second control client could affect the existing client. No socket
messages were sent. Do not use a messaging endpoint just to monitor a session.

The public [hook reference](https://code.claude.com/docs/en/hooks) applies across
terminal, IDE, Desktop, and cloud clients and offers permission and lifecycle
events. This is an optional route to permission detail, not a baseline dependency.
Cowork's separate `local-agent-mode-sessions` tree was inspected for layout only;
ordinary Chat/remote conversation collection remains future work.

### Grok Build

Installed version `1.0.46`. Local documentation ships under `.grok/docs/user-guide/`.
The [CLI reference](https://docs.x.ai/build/cli/reference) also documents session
export; launching a fresh CLI/ACP session is not observing an existing one.

- `.grok/sessions/<percent-encoded-cwd>/<session-id>/summary.json`: `info.id`,
  `info.cwd`, titles, update/activity times, model, `session_kind`, parent ID.
- `updates.jsonl`: ACP-like `session/update` events with `sessionUpdate` values:
  user/agent message chunks, tool calls/updates, and `turn_completed`.
- `events.jsonl`: `turn_started`, phases, tools, permissions, and `turn_ended`.
- `chat_history.jsonl`: complete messages/tool results; sampled rows have **no
  timestamps**. Keep the timestamp unknown unless matched to an update.
- `.grok/active_sessions.json`: session ID, PID, cwd, `opened_at`. Check the
  process executable and lifetime. A reused PID started after the registry entry
  does not validate that entry.

Prefer `params._meta.agentTimestampMs` over an update's outer timestamp: observed
forked/replayed logs have old event times inside newly written rows. Subagents
can inherit parent history; exclude `parent_session_id` and `session_kind`
starting with `subagent`. Headless sessions remain visible and labeled headless;
older nested headless runs without parent metadata cannot be reliably separated.

Question lifecycle observed: `tool_call` with `title=ask_user_question` and
`rawInput.questions`, then cosmetic title updates, then `status=completed` on the
same `toolCallId`. Match by ID, not changing title. A live parent process plus an
unresolved question supports a current wait; a terminal turn clears old calls.
An old incomplete call with no process evidence remains unknown.

`permission_prompt` is not automatically a human wait. Sampled events showed
`permission_requested` immediately followed by `permission_resolved` for routine
allowed tools. The baseline therefore leaves permission state unknown.

**cmux context:** read both turns of **Improve cmux Grok state detection**, thread
`01a0d42f-53e2-7da1-bf5c-643701c804aa`. That investigation found misleading status
in both directions. It proposed missing cancellation/failure hooks initially;
the user corrected the focus to question waits and stale labels after resuming.
Parent/subagent misclassification was a suspect, not a proven cause.

Installed cmux remains `0.64.25` (106). Its hook-fed session list is useful for
surface/workspace identity, but is not treated as current-state authority. The
newer pinned upstream classifier below still combines notification signal and
message text for permission/completion/wait classification. We have not patched
cmux, installed hooks, or claimed a newer release fixes the user's cases.

## Other implementations studied

Source snapshots, not endorsement or proof of live compatibility. Four projects
were cloned; the cmux classifier was fetched at its pinned commit. No external
monitor was installed or executed.

| Project / pinned source | Approach worth reusing | Limit for this task |
| --- | --- | --- |
| [cctop](https://github.com/st0012/cctop/tree/4ee225821cd4951e379f907a517b3a35a2a9b1fc) | Hook helper owns session state; native PID identity; Desktop metadata joins; `CodexDesktopRuntimeProbe.swift` detects app-owned server and SQLite home | Lifecycle depends on hooks; its `Stop` means waiting for input generally, not necessarily an actual question |
| [AgentMonitor](https://github.com/jiweiyeah/AgentMonitor/tree/0d404bdf0968befae73163695342f15e74d1dc04) | Provider adapters, incremental transcript reading, desktop-specific file roots | `infer_status` uses under 60 seconds = active, under 24 hours = idle, older = completed; unsuitable as task completion evidence |
| [agent-dashboard](https://github.com/ProjectNeura/agent-dashboard/blob/d77407910515a3173d746211696d8c59d3ffd15d/src/monitor/codex.rs) | Pair pending Codex request IDs with outputs; clear on terminal events; optional Claude hook bridge | Groups question and escalation calls as approval; needs stronger live ownership and async-question handling |
| [so-agentbar](https://github.com/sotthang/so-agentbar/blob/c839d274eafce3652d6c7b05aa53b7d923c89759/SoAgentBar/CodexSessionMonitor.swift) | Filesystem monitoring, incremental Codex event decoding, session coordinator | Log event vocabulary/version and terminal-state assumptions must be checked against installed clients |
| [cmux classifier](https://github.com/manaflow-ai/cmux/blob/72bdb814562e3f4fa98c917bcf565cbc17379a41/CLI/AgentHookNotificationPolicy.swift) | Hook normalization and terminal surface identity | Notification prose and parent/child routing can disagree with actual question lifecycle |

Adopt the small parsing and identity ideas; don't adopt a monitor's status model
wholesale. Existing monitors answer different questions: stopped inference,
completed turn, unseen output, and question waiting are often mixed together.

## Optional hook layer

Not installed. If needed, write small passive atomic JSON records, keyed by engine,
session ID, parent ID, process identity, turn ID, and call/request ID. Include event
and observation timestamps. No transcript text or credentials are necessary.

Track prompt submission, question open/resolve, permission open/resolve, turn
completion/interruption/failure, and process/session end. Ignore child completion
when updating a parent. Pair request IDs; clear on resume/terminal transitions.
Reconcile against native files and process identity on every one-shot collection;
old hook records do not override newer evidence. Unknown/new events stay unknown.

Grok's installed hook guide distinguishes `Stop`, `StopCancelled`, `StopFailure`,
and delayed `idle_prompt`. It warns that a blocking Stop hook can continue the
turn; a Stop notification alone is not final completion. Test that case before
using hooks for authoritative idle state. Hooks add transition precision; they
must never be required for history, folders, titles, or latest-message collection.

## Validation and next priority

Implemented: filesystem baseline for all three, provider isolation on failure,
source/confidence/timestamps, explicit unknowns, separate activity/question/
permission/process/spec-association fields, and bounded transcript reads.
Latest three sessions per provider/folder are returned, plus registered live
Claude/Grok sessions. This is a recent view, not an exhaustive archival census.

The live ten-checkout scan returned 62 sessions (27 Codex, 11 Claude, 24 Grok)
during validation, with two explicit Claude question waits. Counts and states
change as agents work. The current Codex research thread also appeared as inferred
running when its own folder was requested. No full start/stop transition exercise
was forced on the user's working sessions.

Synthetic tests cover answered/pending/async questions, aborted turns, stale open
turns, PID reuse, partial logs, stream timestamps/chunks, live-session retention,
metadata joins, missing sources, and subagent exclusion. Git inventory tests still
pass. See [development](development.md) for commands.

Next priority: caller-side Codex app-tool enrichment, then a tested read-only
Desktop IPC adapter if standalone live state is worth maintaining. Next for Claude:
permission detail. Next for Grok: approval resolution and cmux surface joins.
Computer use is the final caller-side fallback, following
[the enrichment instructions](agent-inventory-instructions.md).

## Task correlation follow-up (2026-10-02)

The [behavior spec](../.spec/20261002-worktree-task-inventory.md) supersedes the
initial folder-only output. All discovered top-level candidates are now inspected
by default, including archived app metadata. Exact conversation spec references
and unique title-ticket evidence associate sessions; conflicts remain explicit.
Large transcripts use bounded head/tail evidence. Missing local/cloud data remains
unknown. The filesystem formats and activity caveats above still apply.

Read the installed Zoo invocation reference in
`.agents/skills/zoo-spec-uberreview/references/invoke-uberreview.md`: Claude uses
`--no-session-persistence`, Codex `--ephemeral`, and Grok persists its one-shot
reviews. Actual local Grok headless transcripts contain read-only review prompts
and explicit spec references. Those retained reviews are secondary; headless alone
is insufficient. Native child sessions remain excluded by the source collectors.

Live checks found primary Grok chats mentioning another pending spec in the same
worktree. A unique ticket in the chat title corroborates which referenced task
it handles. This remains an inference, and multi-task sessions without that
corroboration stay ambiguous. The user clarified that “stashed” meant “staged”;
the collector now reads and counts only current checkout changes, never stashes.

Claude full scans and incremental updates share status inference. Native `busy`
and an open transcript turn with verified process identity survive silent tool
and subagent waits. Incremental updates reread the native registry, rejecting a
changed PID/start identity until the next full probe. Native input/approval waits,
newer transcript completion, and process exit retain their separate states.
