---
status: implemented prototype; correlation and workflow states are heuristic
---

# Worktree task inventory

## 1. Purpose and boundaries

### User outcome

- Answer what local work exists, who is handling each task, and what needs attention.
- Model **worktree → task (Zoo spec file) → sessions**. Expect at most one active
  primary session per task; expose conflicts instead of silently accepting them.
- Keep one row per unpushed task. Keep an empty worktree visible with a cleanliness
  label when no task qualifies. Pushed history has no rows in this iteration.

### Operating model

- CLI runs once, sequentially. Reusable inventory supports cached per-worktree
  refresh; neither mode uses hooks, network/model calls, or UI automation.
- Default to Prairie `backend`, then `back2`–`back10`. Accept arbitrary paths and
  a replacement parent directory; names are defaults, not permanent identities.
- Read local Git and agent data. Never fetch, alter app state, send prompts, or
  run a review. The caller can enrich unknowns through existing tools or UI.

### Deliverables

- Entry point: `go run ./cmd/talkingduck`. Collection, correlation, and rendering
  run in Go, using modernc SQLite without cgo. Source runs require Git and Go;
  built binaries require Git. No Python runtime.
  `cmd/talkingduck` is a thin entry point; root-level `cli` owns flags/output,
  and `inventory.Build` returns correlated snapshots without printing or exiting.
  Desktop callers can reuse either package; no GUI dependency enters the CLI.
- Default: compact table. `-ai`: unaligned plain text with session IDs. `-details`: legacy
  verbose report. `-no-agents`: Git/spec data with unknown session fields.
- This spec is the behavior contract. Supporting evidence lives in
  [research](../docs/agent-collection-research.md); caller instructions live in
  [enrichment instructions](../docs/agent-inventory-instructions.md).

## 2. Worktree and task discovery

### Unpushed boundary

- Select HEAD commits absent from every local remote-tracking ref. This differs
  from upstream ahead/behind counts, which are a separate display field.
- Include tracked dirty and nonignored untracked specs; include specs touched or
  explicitly referenced by unpushed commits. Deleted specs retain HEAD content.
- Include an unchanged spec when an unpushed commit uniquely matches its ticket,
  or local dirty code uniquely matches paths in the spec. A pushed spec can thus
  describe new local work; session activity alone never resurrects a pushed task.
- Missing remote refs mean unknown history: show local changes and a warning.
  Squashed/cherry-picked equivalents are not detected by commit identity.

### Evidence and identity

- A task is its worktree plus `.spec/*.md` path, including archived paths selected
  by local evidence. A session is provider plus session ID within its worktree.
- Associate commits by direct spec path, then a unique full-message ticket match,
  then the spec's Commits list (hash or historical subject after rebase), then
  shared-ticket matches and file/directory references. Record matching ticket IDs
  as evidence even when another rule selects the owner. Subject matches remain
  inferred; unresolved multiple indirect owners stay unassigned.
- Tickets come from the filename, first request paragraph, explicit Ticket(s)
  lines, and Also include lines. Preserve multiple IDs and background mentions
  separately. Issue-like identifiers can include non-Linear references.
- Use the spec H1 when present; otherwise show its filename slug without date and
  leading numeric ticket prefix. Preserve the original title/path in the inventory snapshot.

### Ordering

- Keep configured worktree order. Within each, rank recent unfinished work,
  older execution, plans, parked work, then done work.
- Recent means seven days. Compare activity times within a group; committed-only
  tasks use Git topological commit order. File mtimes are approximate activity.
- Mistakes logs, proposals, and `.tasks/` files do not promote task activity.
  Session timestamps choose the displayed agent, not task ordering yet.

## 3. Session collection and association

### Local sources

- Codex Desktop: thread SQLite databases, rollout logs, persisted turn history.
  Shared app-server process existence cannot prove an individual turn is running.
- Claude Desktop Code: app chat metadata, linked Claude Code transcripts, native
  process registry. Validate PID, executable, and process start time.
- Grok Build: summaries, message/update/event logs, active-session registry.
  Validate process identity; distinguish event time from fork replay time.
- Scan all discovered top-level candidates by default, including archived app
  metadata. `-agent-limit N` deliberately reduces coverage; registered live
  Claude/Grok sessions survive that limit. Plain-text agent-only listing is available through `-agents-only`; `-home` overrides its
  data-discovery home directory.

### Primary versus secondary

- Observed Zoo invocation rules: Claude uses `--no-session-persistence`; Codex
  uses `--ephemeral`; Grok has no equivalent flag and retains review sessions.
- Exclude explicit native subagents during source collection. Classify a retained
  Grok headless session as secondary when its prompt requests review-only work.
- A top-level Desktop/interactive CLI session is an inferred primary. Headless
  sessions without clear review evidence remain role-unknown. Merely having
  “review” in a title does not disqualify a primary session.
- Preserve associated secondary and historical primary sessions under the task.
  Choose the latest primary by metadata update time across providers. Flag two
  or more apparently running/waiting primaries with `!` after the agent name.

### Correlation rules

- Match exact spec basenames in user/assistant conversation text. Ignore tool
  output and known injected context. References are evidence, not proven ownership.
  Ignore injected memory manifests too. A spec explicitly labeled active/current
  task, or given an explicit executing/implementing status, takes precedence over
  historical references, including in compaction summaries.
  Multiple current-task labels remain ambiguous; later references to another spec
  without a new label invalidate the prior focus. Preserve the matched declaration.
- If several specs are referenced, a unique task ticket in the session title can
  corroborate one. Without spec references, a unique title-ticket match is fallback.
  If neither matches, use tickets in user messages; shared tickets or references
  to several tasks remain ambiguous. Assistant-only ticket mentions do not count.
- Multiple remaining matches are ambiguous. No match is unknown. Never assign
  the only task in a folder merely because it is the only task.
- Association reads transcript head and tail for large files, with a warning.
  The inventory snapshot keeps unassigned sessions at worktree level; table rows remain task-based.

## 4. Workflow and attention

### Task stage

- `--`: insufficient stage evidence or work outside the defined execution stages.
- `HL` / `LL`: high-level / low-level planning, including their spec reviews.
  A later low-level clause takes precedence over earlier high-level approval.
- `EX`: execution. Count top-level `[ ]`, `[>]`, and `[x]` entries only inside
  `Subtasks`; exclude nested checklists and duplicate report summaries.
- `FIN`: final review, or completed listed subtasks awaiting closeout. Generic
  pending-suggestion status with all subtasks done maps here, not to `SQ`.
- `SQ`: squash ran in this revision round, after all subtasks finished, and is
  the latest logged workflow action. Commit count alone never establishes SQ.
  Later execution/review can return to EX/FIN; squash does not prohibit changes.
- `PUSH`: explicit Zoo push invocation or logged push readiness. It is an
  operation/stage observation, never authorization to push.
- `REBA`: standalone Zoo rebase running or just finished. Within Zoo push it stays
  PUSH; within Zoo closeout it finishes as FIN. Read chronological skill context
  from the spec log and explicit invocations/results in associated primary chats.

### Revision rounds

- Display a number immediately before stage: 0 before meaningful work, 1 for
  initial planning/execution, 2 after one major revision request, and so on.
- Infer from chronological Log entries identifying user revision requests, or
  explicit task-round/major-iteration numbers. Review-loop round numbers never
  increment the task round. Routine review findings are not major revisions.
- Starting a revision clears `squashed_this_round`. A squash in round 1 cannot
  produce SQ in round 2; that round needs its own completed squash.
- Preserve matched revision entries, latest logged action, and inference confidence
  in the inventory snapshot. Missing or loosely worded log history can undercount revisions.
- Future Zoo frontmatter may publish round, stage, operation context, and squash
  round explicitly. This collector does not modify source specs or require it.

### Progress and interaction

- Progress `▶`: latest primary has evidence of running. `✓`: spec says stage
  ready/approved/awaiting action, or a logged operation completed. `?`: unknown.
- Interaction blank: no outstanding question detected. `!`: primary is waiting
  or has an observed unresolved question. `?`: uncertain. These are separate
  from progress; a question can interrupt work before a stage is ready.
- Show subtask completion as `done/total`, including `0/0` before subtasks exist.
- Show question/suggestion counts as `Q/S`. Questions count unresolved structured
  prompts; prose-only uncertainty displays `?`. Suggestions count top-level open
  items under Pending suggestions, ignoring templates, nested detail, and checked
  items. Unstructured suggestion prose has unknown count.

### Evidence limits

- Native Claude waiting alone does not distinguish question from permission.
  Grok permission phases may resolve automatically; permission stays unknown.
- Pair structured questions and answers by tool-call ID. Async question tool
  return is not an answer. Preserve unresolved prompts with their source.
- Open persisted turns without fresh events become unknown. A completed turn
  does not prove the Zoo task is done. A sidebar badge alone proves neither.
- Agent activity and spec status can disagree; preserve both. Never manufacture
  approvals, completed reviews, or a current question from file recency alone.

## 5. Git display

### Local counters

- `±N`: unique tracked modified paths, staged/unstaged, including
  Markdown. Rename source bookkeeping does not count as another current file.
- `?N`: untracked paths excluding informational Markdown.
- `ℹ︎N`: changed/untracked Markdown under `.spec`, `.proposal`, `.proposals`, or
  `.tasks`. Tracked informational Markdown also counts under `±`, intentionally.
- Count each path once even when both staged and unstaged. Never read or count
  stashes; they are outside this inventory.

### Remote and empty-worktree state

- `⇣N ⇡N`: commits behind/ahead of the configured upstream using local refs.
  Missing/unreadable upstream produces `⇣? ⇡?`. Never fetch automatically.
- Repeat worktree counters on every task row; they are not task-specific totals.
- For a worktree without selected tasks, show `CLEAN` when all changes are
  informational Markdown; `MARKDN` when all changes are other Markdown;
  otherwise `DIRTY`. No changes also means `CLEAN`. Classify current checkout
  changes only; saved stashes have no effect.
- A failed Git observation must remain visible as an error or warning; do not
  interpret an unavailable checkout as clean.

## 6. Output contract

### Compact table

- No headers or borders. Columns, in order:
  - Worktree, latest primary agent, ticket(s), revision round, task stage.
  - Subtasks done/total, progress, interaction, questions/suggestions.
  - Local Git counters, remote Git counters, spec title.
- Use one space between padded columns. Fix widths across the table, expanding
  only enough to fit observed values. Unicode symbols occupy terminal cells.
  Give each Git counter (`±`, `?`, `ℹ︎`, `⇣`, `⇡`) its own aligned column.
  Reserve at least two digits for each number, including rounds and paired counts.
- Last column uses remaining width and wraps. Use `-width N`, otherwise terminal
  width / `COLUMNS`, otherwise 120. Below the structural minimum the table can
  exceed the requested width; never silently clip identifiers or counters.
- Multiple tickets occupy separate physical lines in one cell. Other cells are
  top aligned. Long titles may also increase row height.
- Color terminal output automatically; keep pipes plain. `-color always|never`
  overrides detection; auto honors `NO_COLOR` and `TERM=dumb`. AI output stays plain.
  Zero counts and unknowns are gray; agents, stages, and nonzero counter columns
  have distinct colors. Attention is yellow, conflicts/errors red, titles white.
  Apply ANSI escapes after layout so colors cannot change alignment or wrapping.

### Cached inventory

- `inventory.New(options)` creates pending entries without scanning. `Paths()`
  returns normalized, deduplicated worktrees in configured order.
- `Update(path)` scans one configured worktree and its scoped sessions. Shared
  provider indexes may still be read; no other worktree is rescanned or cleared.
- `Snapshot()` returns detached cached data without collection or waiting for a
  refresh. In-memory copying scales with data size. No caller mutation leaks back.
- Track each row's pending/updating/ready/error state, successful observation time,
  latest attempt/completion times, and refresh error. Initially unknown, never clean.
- Publish each refresh atomically. Failures retain last successful data and its
  timestamp, expose the error, and fail `Snapshot.Failed()`. Source warnings remain
  nonfatal. Top-level observation time is the newest row, not all rows' freshness.
- Same-worktree refreshes serialize; other worktrees can refresh independently.
  Cached running/attention states must be interpreted using per-row timestamps.
- GUI/MCP callers own scheduling and stale-age policy. No timer, disk persistence,
  singleton, or GUI/MCP wiring is added. CLI retains one-shot `Build` behavior.

### Agent output and diagnostics

- `-ai` uses full worktree paths, comma-separated tickets, no padding/color/wrap,
  and `spec=`/primary `session=` references. It overrides width/color/details.
  Empty status cells use `-`; finish with one line explaining columns and symbols.
- No CLI JSON mode. The inventory API retains schema version 2: `repositories[].specs[].sessions`, `primary_session`,
  `primary_conflict`, `workflow`, and worktree `git` observations.
- Preserve raw Git/spec evidence, unassigned files/commits/sessions, session
  findings, source timestamps, inference reasons, and collection warnings.
- Source failures degrade agent coverage without hiding Git work. Checkout
  failures produce a nonzero exit. Correlation-helper failure also fails visibly.
- Live output contains personal chat excerpts/questions. Keep snapshots ignored;
  fixtures must be synthetic. Plain table operational diagnostics go to stderr.
- Ambiguous commit ownership is structured data, not one warning per hash.
  `commit_association_issues` includes subject, rule, candidate specs, and matched
  paths. A shared ticket does not prove which phase owns newer work.

## 7. Validation and next steps

### Required checks

- Run inventory tests and build with `CGO_ENABLED=0`; run `go vet`,
  `git diff --check`, and documentation link checks. Synthetic tests cover
  read-only SQLite access, including uncheckpointed WAL data.
- Cover pushed-task exclusion, dirty/untracked discovery, ambiguous ownership,
  secondary reviews, primary selection/conflict, counts, stash exclusion,
  upstream failures, multi-ticket alignment, and empty worktrees.
- Exercise the real local scan read-only. Compare selected matches and secondary
  classifications against transcript evidence; retain uncertainty in the output.
  The Go port reproduced the previous compact table and compared 1,082 local
  session identities against the pre-port snapshot.

### Known gaps

- Cloud-only Codex chats and ordinary Claude Chat/Cowork remain outside coverage.
  Missing local transcript data can leave an otherwise obvious task unmatched.
- Bounded transcript evidence can miss older references or find background
  references. Sessions reused for multiple tasks can remain ambiguous.
- Log-based stage/round classification, suggestion formatting, title-ticket parsing,
  and seven-day ordering need refinement from use. No explicit approval/round ledger exists.
- Snapshots are non-atomic and local remote refs may be stale.

### Optional enrichment

- Prefer existing app tools/private read APIs for unknowns, then read-only UI
  inspection by the calling agent. The collector never recursively invokes one.
- Hooks may later add explicit ownership, role, stage, question lifecycle, and
  parent IDs. They remain a bonus layer; filesystem collection stays useful alone.
- Durable history, recent pushed tasks, interactive control, and app shape remain
  future work. This experiment does not select a UI framework or service.
