# Event-driven refresh

The desktop has independent cheap and expensive queues. Event sources mark work
pending; they do not collect inventory. No routine five-second filesystem poll or
unconditional one-minute inventory pass remains.

| Queue | Cooldown: quiet / maximum wait | Throttle after processing finishes |
| --- | --- | --- |
| Cheap | 1 second / 5 seconds | 5 seconds per worktree |
| Expensive | 5 seconds / 20 seconds | 60 seconds per worktree |

For a pending batch, eligibility is the later of the throttle deadline and
`min(last event + quiet period, first event + maximum wait)`. A busy worker can
add queueing delay. Events during processing form the next batch; finishing does
not clear them. Each queue has one worker. A slow expensive scan cannot occupy
the cheap worker. Manual/MCP refreshes use the expensive queue too; MCP waits for
its requested batch, subject to request cancellation.

## Event sources

- **Native filesystem:** fswatcher v0.1.0, FSEvents through purego on macOS.
  Watch worktrees recursively, plus actual Git and shared Git directories resolved
  through `.git`/`commondir` files. Source/Git changes request expensive work;
  spec changes request both queues. Watch provider roots for session/transcript,
  registry, SQLite database and WAL changes. Ignore SQLite SHM notifications.
- **Provider routing:** filesystem events request a coalesced cheap metadata check.
  Existing `ChangeProbe` maps changed evidence to worktrees. Shared SQLite changes
  query thread metadata; cached JSON/file metadata avoids full history parsing.
  Affected folders then enter both queues. This discovery step can add latency
  before the per-folder cooldown; no idle polling of provider files remains.
- **Process liveness:** every 30 seconds, compare identity-checked Claude/Grok
  registry processes. Changed identity/liveness (including removed records) emits
  the same per-folder expensive event. Observation timestamps alone emit nothing.
  Codex has no equivalent per-chat PID evidence in the current collector.
- **Recovery:** startup, watcher errors/dropped-event reports, manual refresh, and
  a ten-minute timer request full reconciliation through the same event path.
  The recovery timer also recreates watches to repair replaced roots. If creating
  the native watcher fails entirely, warn and fall back to minute reconciliation.
- **Fetch:** launches independently at startup and every 15 minutes, up to five
  concurrent fetches. Ref/index/filesystem changes return through native events;
  completion does not directly trigger collection. Failed fetches remain warnings.

File notifications are hints, not Git status or proof of agent activity. Directory
replacement and event loss can require recovery. Unsupported provider schemas and
inaccessible data remain warnings/unknowns. No hooks or recursive agent calls.

## Collection

Cheap work reparses known specs against cached Git selection and ownership, then
reads appended complete transcript lines with incremental cursors. It updates
workflow, title, tickets, questions and transcript-derived activity. Initial/reset
reads use a 256 KiB tail. Truncation/replacement resets the cursor. No Git or `ps`
runs in the cheap queue.

Grok open-turn evidence persists through quiet tools/reasoning when its current
registered process is verified alive. Full and incremental collection share the
same lifecycle reducer. A later turn completion or stopped process ends running;
output age alone does not. When registry entries share a PID, only the newest
registration can claim that live process.

Expensive work discovers tasks/sessions, checks Git cleanliness and commit
association, reassesses ownership/process evidence, and updates retained pushed
history. New/deleted tasks and native registry status can therefore wait for the
expensive throttle. Running process checks is an independent event source; its
changes do not directly mutate cached activity.

Collection already sets `GIT_OPTIONAL_LOCKS=0`, preventing optional index refresh
writes by `git status`. TalkingDuck history is outside default watched worktrees
in Application Support. Fetch intentionally writes Git data. Custom state-file
placement inside a watched worktree would create feedback and should be avoided.

## Synchronization and checks

Full refreshes serialize; cache publication briefly shares a lock with cheap
updates. Full collection does not hold that publication lock. Before publishing,
a full result refreshes known specs/session evidence again to avoid undoing newer
cheap updates. Snapshot generations reject old UI publications. Cached reads stay
immediate and detached.

Queue tests cover exact deadlines, continuous-event caps, completion-based
throttling, coalescing, events during a blocked job, independent cheap work, and
cancellation. Native tests cover spec events and external Git directories. Process
tests cover timestamp noise, liveness changes, removed records and folder scoping.
The [native diagnostic](../cmd/watchcheck/README.md) passed all ten Prairie folders
without cgo; it does not diagnose another app's watcher or guarantee lossless delivery.
