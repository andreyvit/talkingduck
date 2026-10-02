# Development

## Current state and layout

The inventory CLI and macOS menu bar app share one root Go module.
The desktop prototype has a local app build, but no release/notarization process.
The GUI window displays live inventory through the CLI's shared table rows.
The menu bar image still uses mock data.

- `.agents/skills/`: Zoo 3.2, `terse`, `linus`, and the local `commit` skill.
- `.codex/agents/`, `.claude/agents/`: installed Zoo roles.
- `.claude/skills`: symlink to shared skills.
- `.zoo/`: project workflow configuration.
- `docs/`: durable AI and developer context.
- `cmd/talkingduck/`: thin cgo-free CLI entry point.
- `cli/`: reusable flag registration, terminal output, and exit codes.
- `desktop/`: Wails app, native tray-width shim, embedded overview, and local build script; see [desktop instructions](../desktop/README.md).
- `menubarimage/`: shared standard/compact pixel renderer used by the app and scratch command.
- `inventory/`: read-only builder, correlation, and in-memory per-worktree cache; modernc SQLite without cgo.
- `cmd/scout/`: small standard-library Go review script; no module needed.
- `.spec/YYYYMMDD-taskslug.md`: tracked task specs.
- `.proposals/YYYYMMDD-title.md`: tracked proposals; archive in `.proposals/archived/`.
- `.tasks/YYYYMMDD-taskslug-research.md` and
  `.tasks/YYYYMMDD-taskslug-evidence/`: ignored research and evidence.
- `.ai-mistakes.md`: brief lessons from corrected agent decisions.

Create task and proposal directories when needed. Do not populate them with
placeholder plans.

## Working conventions

Build the smallest useful stage that addresses the current request. Follow
existing code once it exists; add architecture only for demonstrated needs.
Keep factual observations separate from inferred state, including uncertainty
and source age when collecting agent information.

Keep personal paths, ticket histories, transcripts, and credentials out of
fixtures; use synthetic examples. Read-only monitoring is a sensible initial
experiment, but product actions require a separate concrete design.

Use the local `commit` skill for commits. Review staged content, keep changes
coherent, and describe the purpose. Push only when requested. `desktop/build.sh`
creates an ad-hoc-signed local app;
no distribution release or deployment commands are defined.

## Validation

For scout changes, run its own tests:

```sh
go test cmd/scout/scout.go cmd/scout/scout_test.go
```

For documentation and setup changes, check links, installed copies, symlinks,
ignore rules, and `git diff --check`. For future code, add relevant validation
commands here when the implementation provides them.

The scout prints questions for a reviewer; it does not verify the product or
run agents. See [its design](../cmd/scout/DESIGN.md). While editing the scout,
follow [its instructions](../cmd/scout/AGENTS.md).

For inventory changes, follow [its collection rules](../cmd/talkingduck/README.md)
and run:

```sh
CGO_ENABLED=0 go test ./inventory ./cli ./cmd/talkingduck
go vet ./inventory ./cli ./cmd/talkingduck
# For cache/concurrency changes:
go test -race ./inventory
CGO_ENABLED=0 go build -o /tmp/talkingduck ./cmd/talkingduck
```
