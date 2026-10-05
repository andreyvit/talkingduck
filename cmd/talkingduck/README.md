# Local work inventory

Terminal UI and one-shot Go collector; reads projects and keeps its own bounded task history. Uses modernc SQLite without cgo.
No Python runtime or service.

```sh
go run ./cmd/talkingduck          # default: TUI
go run ./cmd/talkingduck -status  # one-shot table
# Token-efficient plain text for agents, including primary session IDs:
go run ./cmd/talkingduck -ai
# Override the default ~/Developer/prairie parent:
go run ./cmd/talkingduck -root /path/to/checkouts
# Or supply any checkout paths explicitly:
go run ./cmd/talkingduck /path/to/one /path/to/two
```

Modes are shared with the desktop executable: `-tui`, `-gui`, `-status`, `-ai`.
Only their defaults differ (TUI here, GUI there); explicit modes are mutually
exclusive. `-details` and `-agents-only` remain one-shot modes. `-width` and
`-color` control one-shot output; the TUI follows the terminal size and palette.
`-mcp-port` applies only to GUI mode.

The TUI keeps a cached inventory, watches native events, and runs independent
cheap/full queues with the same cooldown and throttle policies as the desktop.
It does not fetch remotes. `r` queues the selected worktree; `R` queues all.
Up/Down selects. Open directly with `a` agent, `t` ticket, `s` spec, `f` folder.
Only `c` starts a chord: `ct`, `cs`, `cf` copy ticket numbers, spec titles, and
full folder paths. Multiple tickets get an Up/Down/Enter chooser. `g` is reserved
for future Git opening. Esc cancels; Shift-Q/Ctrl-C exits.
Mouse: click rows to select, links to open, or hover and click `©` to copy.
The wheel scrolls vertically; keyboard selection brings its row back into view.
Titles wrap; horizontal scrolling is disabled.

Defaults to `backend`, then `back2` through `back10`, scanned sequentially.
Paths are configuration, not product identities. Separate worktrees stay separate.
Requires Git and Go to run from source; a built binary needs Git.
`-no-agents` skips session collection.
Recent local tasks are remembered in `~/Library/Application Support/TalkingDuck/state.json`
on macOS. One task per worktree stays visible as `PUSHED` after leaving local
selection. Pulled-only tasks do not qualify. Use `-state-file PATH` to relocate
history or `-state-file ''` to disable it. See [retention rules](../../inventory/README.md#recently-pushed-tasks).
Does not fetch, write Git state, run agents, install hooks, or contact tickets.
A failed checkout appears in the report; other checkouts continue. Exit status is
nonzero if any checkout fails. Each Git command has a 30-second timeout.

## AI output

`-ai` prints the table values without alignment, colors, or wrapping, even with
`-color always`. Each row uses the full worktree path; multiple tickets share a
comma-separated cell. Task rows include `spec=path` and the latest primary
`session=ID` when known. Empty progress/input cells become `-`. The final line
explains columns, symbols, and stage codes. `-ai` takes precedence over `-details`.

`-json` was removed. Full structured evidence remains available through
`inventory.Build`; the CLI does not serialize it. `-agents-only` lists raw session
identities, activity/question states, source times, and coverage warnings in text.

## Compact task table

The default is a headerless table, one row per worktree/task. Columns are:
worktree, latest primary agent, tickets, revision round, stage, subtasks, progress, interaction,
question/suggestion counts, local Git counters, upstream counts, title.
The `±`, `?`, `ℹ︎`, `⇣`, and `⇡` counters each have their own aligned column.
Every number reserves at least two digits, including rounds and paired counts.
Colors are automatic in terminals; pipes and AI output stay plain. Use `-color always`
or `-color never` to override; auto honors `NO_COLOR` and `TERM=dumb`.
Zeros/unknowns are gray. Codex is cyan, Claude orange, Grok purple; stages have
distinct colors. Nonzero tracked/untracked/info/behind/ahead counts are
yellow/green/blue/red/cyan. Attention is yellow; the trailing title stays white.
Tickets use separate lines; remaining cells are top aligned. The title wraps into
remaining width. Use `-width 150` to choose a width; `-details` keeps the old report.

Stages: `-- HL LL EX FIN SQ PUSH PUSHED REBA`. SQ means squash ran in this round;
a new revision resets it. REBA means standalone Zoo rebase; push/closeout retain
PUSH/FIN. The spec Log and explicit primary-chat skill invocations supply evidence. `▶` means working,
`✓` ready/awaiting the next action, `!` interaction needed, `?` unknown.
`±` counts tracked staged/unstaged modified paths, `?` untracked noninformational paths,
`ℹ︎` informational Markdown. Tracked informational files count in both ± and ℹ︎.
`⇣`/`⇡` compare the local upstream ref. Counters and CLEAN/DIRTY describe only
current checkout changes. Stashes are never read or counted.

Sessions nest under specs. Persisted Grok review-only headless sessions are
secondary; primary selection uses the latest associated primary's timestamp.
An agent name ending in `!` signals multiple apparently active primaries. Unknown
matches remain unassigned, never attached by folder or spec recency alone.
When spec references and title tickets do not match, a unique ticket match from
user messages can associate the session. Shared tickets remain ambiguous;
tool output, injected context, and assistant-only ticket mentions do not count.
In reused or compacted chats, an explicit active/current-task label attached to a
spec path, or an explicit executing/implementing status, takes precedence over
historical references. Conflicting labels stay
ambiguous; later discussion of another spec invalidates that focus. Injected
memory manifests do not establish ownership.

Current Zoo frontmatter takes precedence: `revision_round` (initial round 0),
`stage` (including INV), `stage_complete`, and `tickets` (including an empty list).
Parsing uses restricted regular expressions, not a YAML dependency. Unsupported
or duplicate fields fall back to legacy inference; older round counts remain
inferred. H1 supplies the task title.

Standalone assistant receipts identify primary/review ownership and current-round
squash, rebase, and push operations. Push/closeout rebase context is preserved;
start/failure does not imply completion. Fenced examples, quoted lines, user
instructions, and other-task/other-round operations do not drive those overlays.
Receipts do not prove runtime activity. Patchset formatting alone does not mean SQ.

Grok can retain a saved `subagent` label after interactive resume. A verified
live registration for the same session and folder overrides that label, with
the discrepancy recorded. Explicit parent sessions remain excluded.

See the [single behavior spec](../../.spec/20261002-worktree-task-inventory.md)
for selection, status rules, evidence limits, and the complete output contract.

## What it collects

- Branch, HEAD, unpushed commits, staged/unstaged/untracked paths.
- Specs changed by those commits or local changes, including deleted specs.
- An unchanged spec when an unpushed commit explicitly names it or uniquely
  matches its ticket; an unfinished pushed spec when dirty code uniquely matches
  its paths. Completed specs need explicit local spec/commit evidence to reopen.
  Candidates with no owned local changes are discarded.
- Raw frontmatter `status`, a conservative stage label, likely task tickets,
  request-section ticket mentions, and timestamp evidence.
- Unassigned commits/files and auxiliary changes. Mistakes logs, proposals, and
  `.tasks/` changes do not promote a spec.

“Unpushed” means reachable from HEAD but absent from **all local remote-tracking
refs**, including feature branches. This uses commit identity, not patch identity;
cherry-picked/squashed equivalents can still appear. No remote refs means unknown
history, not that the entire repository is new. The report then uses local changes
only and warns. Remote knowledge may be stale; fetch separately when desired.

Discovery reads tracked and nonignored untracked Markdown under `.spec/`, including
archives. Historical specs are omitted unless local evidence selects them.
If no spec is found but commits remain unassigned, the worktree row explicitly
shows the unpushed commit count and `spec unknown`, alongside file cleanliness.
Rename sources and destinations are reported separately for now.

## Associations and uncertainty

Direct spec changes are evidence. For other commits, prefer a spec path in the
full commit message, then a unique spec-ticket match from the full message, then
hashes listed under the spec's Commits heading. After a
rebase, match a unique current subject to a saved subject or the resolved old commit
object's subject from that list. These subject links are explicit inferences.
Shared-ticket matches can be disambiguated by that historical commit evidence;
otherwise they remain ambiguous. Referenced paths are the final fallback. Every
assigned commit also records each ticket it shares with its owning spec, even
when another rule established the association. Explicit
backticked directory references with at least two components also match files
below them. Multiple indirect matches stay unassigned; multiple directly changed
specs can share a commit. Dirty files use spec paths or files in associated commits.
There is no “only spec in the folder, therefore owner” shortcut.

Unresolved multi-spec commit matches are `commit_association_issues` in the inventory snapshot:
commit hash and subject, the matching rule, candidate specs, and overlapping paths.
They do not emit hash-only stderr warnings. Operational failures still do.
Shared tickets or directories alone cannot distinguish an umbrella plan from an
implementation phase; later work without an updated phase spec stays unassigned.

Path and ticket associations are inferences, exposed through `inventory.Build`. A referenced path
can be background material, so these links are useful hints, not ownership proof.
A spec that has no local changes and no matching explicit commit reference will
not be discovered from unpushed code paths alone. Unassigned work remains visible.

Tickets come from the filename, the first request paragraph, `Ticket:`/`Tickets:`
lines, and `Also include` lines. This supports multiple tickets and keeps later
background references out of the primary list. Other formats may be missed;
`mentioned_tickets` retains request-section mentions for inspection. No ticket
API is used. The title is the first H1 or the filename without its extension.

Stage is inferred from raw status, not from whether an agent stopped or a commit
exists. `completed subtask 24` means execution, not done. Unknown prose remains
unknown. A stale status can disagree with current work; both remain visible.

## Initial ordering

Within each checkout:

1. Unfinished, nonparked specs with activity in the last seven days.
2. Older unfinished specs with commits, associated dirty code, or execution status.
3. Other unfinished plans.
4. Explicitly parked, deferred, blocked, or waiting-for-dependency specs.
5. Completed specs.

Within each group, use newest activity. For two committed-only specs, use their
latest associated commit's order in Git's topological log, which also handles equal
rebase timestamps. Dirty spec/code activity uses filesystem mtime; committed
activity uses committer time. These are approximations, not a claim about the last
human interaction. Copying files or restoring a stash can change mtime. Auxiliary
changes do not affect ordering. The seven-day rule is an initial heuristic.

The inventory snapshot includes observation time and HEAD. A changing HEAD or Git status
during scanning produces a warning. This is not an atomic snapshot: edits that
leave the same porcelain status can still happen mid-scan.

## Agent sessions

Agent collection runs by default. `-agent-limit N` chooses recent sessions per
provider/folder (default 0 = all), plus registered live Claude/Grok sessions. The inventory snapshot has
per-task `sessions`, worktree `unassigned_sessions`, top-level `agent_sources`, and source warnings. A failed
agent helper sets `agent_error` while preserving the Git inventory. An unavailable
provider does not hide the others. Git errors still produce a nonzero exit;
agent coverage warnings are data, not a fatal exit.

Each session reports client, ID, title, folder, message excerpt/time, activity,
question, permission, process, and spec association. State fields include source,
time, confidence, and reason. `unknown_fields` tells the caller what remains open.
Folder association does not establish spec ownership. Source timestamps are UTC.
Permission remains unknown. Spec references and unique ticket matches provide inferred ownership.

Codex reads local SQLite/rollouts; Claude Desktop Code reads app metadata,
transcripts, and its native process registry; Grok reads summaries, transcripts,
events, and its active registry. No network, UI automation, hooks, or model calls
run inside the collector. It uses modernc SQLite in read-only/query-only mode (including live WAL data) and
bounded 4 MiB log tails. Agent collection builds with `CGO_ENABLED=0`.
Partial or truncated logs are reported; private schema changes can reduce coverage.
Recent-event activity is an inference; silent open turns become unknown after two
minutes. Claude's validated native process status does not expire just because a
question has waited a long time.

See [research and limits](../../docs/agent-collection-research.md) and
[caller enrichment instructions](../../docs/agent-inventory-instructions.md).
Cloud-only Codex tasks and ordinary Claude Chat/Cowork are outside this baseline.
Archived metadata is included when present; missing archived transcripts remain unknown. Live output is personal data; don't commit it as a fixture.

## Validation

Synthetic repositories cover discovery, untracked specs with committed code,
pushed history, missing remote refs, ambiguous ownership, dirty code matching a
pushed spec, rename/deletion, sorting, ticket parsing, and checkout failures.

```sh
CGO_ENABLED=0 go test ./inventory ./cli ./cmd/talkingduck
go vet ./inventory ./cli ./cmd/talkingduck
CGO_ENABLED=0 go build -o /tmp/talkingduck ./cmd/talkingduck
git diff --check
```

Keyboard: `cT` copies the ticket URL; `ct` copies its number. Multiple tickets
use the same chooser.

Shift-X archives the effective row permanently in the state file (GUI and TUI).
Task rows are identified by worktree/spec; a task-free row is archived separately.
No Git files are changed. Running tasks stay archived too.
