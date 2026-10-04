# Native watcher diagnostic

Test recursive filesystem events through `github.com/fswatcher/fswatcher` v0.1.0.
macOS uses FSEvents through purego; cgo is not required. The library also supplies
Linux, Windows, and FreeBSD backends; those have not been tested here.

```sh
CGO_ENABLED=0 go run ./cmd/watchcheck -probe \
  ~/Developer/prairie/backend ~/Developer/prairie/back{2..10}
```

Explicit `-probe` and folder arguments are required. Registers all roots first,
then creates a unique `.talkingduck-watchcheck-*` subtree in each. Checks creation,
editing, renaming, and deletion, including a nested directory created after watch
registration. Existing files are not edited. Each step waits up to five seconds
(`-timeout` overrides this). Temporary subtrees are removed on completion/failure
or Ctrl-C; a forced kill can leave them behind. Nonzero exit means setup failed or
an expected event was missed. Concurrent unrelated events are discarded.

This is an event-delivery diagnostic, not a Git-status implementation. The desktop uses the same library for
[event-driven refresh](../../docs/live-refresh.md). Running this diagnostic does
not change its schedule. Events can be
coalesced: the test checks expected flags and paths, not exact event counts.

## Observed on 2026-10-04

With `CGO_ENABLED=0`, all ten Prairie worktrees passed. The slowest tested event
per folder arrived in 12–15 ms. This confirms recursive FSEvents delivery for new
probe files on this Mac during the test. It does not establish why Sublime Merge
fails to refresh, whether its watch stream is healthy, or delivery under sustained
load, sleep/wake, overflow, or unusual filesystem conditions.
