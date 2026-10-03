# Live refresh

The desktop runs two independent loops:

- Every five seconds, probe change signals and cheaply refresh affected folders.
- Keep the existing sequential full scan, followed by a minute of rest. Manual/MCP
  full scans share its scan lock. A folder never runs two full scans concurrently.

Cheap updates run even while a full scan of the same folder is blocked. No Git,
`ps`, network calls, or provider-wide transcript parsing occurs in this path.

## Signals

Poll file metadata under `.spec`, not the source tree. Poll Claude Desktop/session
metadata and linked transcript metadata; Grok summaries, registry, and event/update
files; Codex database/WAL metadata and the latest 20 thread rollouts per folder.
Read JSON metadata only when its file changes. Query Codex thread metadata only
when its database/WAL changes; compare per-folder results to avoid waking other
folders. SQLite connections are short-lived and read-only.

This is a hint detector. Unsupported schemas and inaccessible sources leave the
periodic full scan as fallback. No hooks or filesystem-watcher dependency.

## Cheap updates

`Inventory.UpdateLive(path)` reparses known task specs against cached Git selection
and ownership. It updates title, tickets, workflow stage, completion, and counts.
Known task sessions use incremental JSONL cursors: read appended complete lines,
retain partial final lines, and reset on truncation/replacement. Initial/reset
reads use a 256 KiB tail. Existing provider reducers retain state across appends;
Grok lifecycle events and new receipts are included. No full-history reparse.

New task/session discovery, ownership reassignment, Git cleanliness/commit
association, and process checks remain on the full scan. A disappeared spec keeps
its last result until discovery reconciles it. The cheap path does not claim those
observations became fresh. `cache.live_observed_at` is separate from full-scan
`cache.observed_at`. Transcript-derived activity remains inferred.

## Synchronization

A per-folder full-scan lock prevents duplicate expensive work. A separate brief
publication lock protects cheap updates and the final full-scan publication;
expensive collection never holds it. Before publishing a full result, refresh its
known specs/session evidence again so an old scan cannot undo a newer status.

Snapshots carry monotonic cache generations. UI publication is serialized and
rejects older generations, including tray updates. Readers keep receiving stable,
detached snapshots during either kind of update.

Measured on backend during development: Git/spec scan about 155 ms; agent
collection 2.37 s; repeated cheap updates about 7–8 ms. Two-folder signal checks
about 120 ms after warmup. These are observations, not latency guarantees.
