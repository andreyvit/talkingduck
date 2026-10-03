# Inventory builder and cache

`inventory` owns read-only Git/spec/session collection, association, workflow
inference, and shared results. It has no flags, terminal formatting, stdout/stderr
writes, process exit, Wails import, or cgo requirement.

```go
snapshot, err := inventory.Build(inventory.Options{
    Paths: []string{"/path/to/checkout"},
    // NoAgents: true,
})
if err != nil {
    // Invalid options or an unrecoverable build error.
    return err
}
// snapshot.Overview: full schema-versioned worktree > task > session JSON data.
// snapshot.Details: legacy typed Git/raw-session report for -details.
// snapshot.Failed(): at least one checkout failed; other rows remain available.
```

Empty `Paths` selects `backend`, then `back2`–`back10` under `Root`, defaulting to
`~/Developer/prairie`. `AgentLimit: 0` includes all discovered top-level sessions.
`CollectSessions(home, paths, limit)` supplies raw provider observations without
Git or spec correlation; an empty home uses the current user's home directory.
Provider environment overrides still apply.

`Overview` retains the prototype's dynamic JSON evidence schema (`Object` is
`map[string]any`); unknown provider fields remain explicit. The existing
[behavior spec](../.spec/20261002-worktree-task-inventory.md) defines the payload.
Each call returns fresh data owned by its caller. Do not mutate a snapshot while
another goroutine reads it. Collection is sequential with per-command/database
bounds; desktop callers should invoke it off the UI thread. `Build` remains the
one-shot API used by the CLI.

Terminal output belongs in [cli](../cli/README.md). Both entry points can reuse
that package; GUI views can consume `snapshot.Overview` directly.

## Cached inventory

```go
cache, err := inventory.New(inventory.Options{Paths: paths})
if err != nil { return err }

// Immediate, detached cached data; no collection. Initially all folders are pending.
snapshot := cache.Snapshot()

// Run off the UI thread, one folder at a time on the caller's chosen schedule.
for _, path := range cache.Paths() {
    if err := cache.Update(path); err != nil {
        // Log/display failure; the previous successful data remains in the cache.
    }
    snapshot = cache.Snapshot()
    // Publish snapshot to the UI after each folder, without waiting for the rest.
}
```

`New` normalizes/deduplicates configured absolute paths without scanning them.
`Update(path)` rejects unknown paths and collects only that worktree's Git/specs
and scoped agent sessions. Shared provider indexes may still be read to discover
those sessions. It atomically replaces that folder after the scan; it never
rescans or clears other folders. Scheduling belongs to the GUI/MCP caller: the
cache starts no timer or background goroutine and persists nothing to disk.

Each repository has `cache` metadata in both snapshot forms:

- `state`: `pending`, `updating`, `ready`, or `error`.
- `observed_at`: last successful collection start; zero until first success.
- `attempted_at` / `updated_at`: latest refresh start / completion.
- `error`: last failed refresh; retained while retrying, cleared on success.

Pending folders have explicit unknown/error placeholders, never a clean result;
`Snapshot.Failed()` remains true until every folder has a result.
A failed refresh returns an error and preserves prior successful tasks/sessions
with their original observation time. `Snapshot.Failed()` also detects refresh
errors. Partial agent coverage publishes normally with provider warnings.
Cached state is not live state: callers must use each row's metadata when showing
running/attention indicators. Top-level `observed_at` is only the newest row's
successful scan start. Provider sources are also retained per worktree; combined
sources carry their worktree and observation time.

Reads never wait for collection. They briefly capture cache references, then
copy data in memory so caller edits cannot affect the cache. Copy cost scales
with snapshot size. Updates to the same folder serialize; different folders may
refresh concurrently. Returned snapshots are internally consistent and stable
after later updates. There is no global cache singleton or `os.Exit` in this API.

`UpdateLive(path)` updates known specs and incrementally reads known session logs
against cached Git/ownership. It runs concurrently with full collection, invokes
no subprocesses, and sets `cache.live_observed_at` without changing Git freshness.
It returns false before a full baseline exists. Snapshot generations prevent stale
UI publication. See [live refresh](../docs/live-refresh.md) for signals, locking,
incremental parsing, and discovery limits.
