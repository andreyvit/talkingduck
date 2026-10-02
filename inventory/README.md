# Inventory builder

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
bounds; desktop callers should invoke it off the UI thread. There is no shared
cache, polling loop, or app startup in this package.

Terminal output belongs in [cli](../cli/README.md). Both entry points can reuse
that package; GUI views can consume `snapshot.Overview` directly.
