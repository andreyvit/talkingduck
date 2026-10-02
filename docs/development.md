# Development

## Current state and layout

The first product experiment is a one-shot Git/spec inventory. A root Go module manages dependencies. No
Xcode project, application build, browser harness, or release process exists yet.
Do not report app tests as passing.

- `.agents/skills/`: Zoo 3.2, `terse`, `linus`, and the local `commit` skill.
- `.codex/agents/`, `.claude/agents/`: installed Zoo roles.
- `.claude/skills`: symlink to shared skills.
- `.zoo/`: project workflow configuration.
- `docs/`: durable AI and developer context.
- `scripts/inventory/`: read-only Git/spec/session collector; Go reads agent files and SQLite with the cgo-free modernc driver.
- `scripts/scout/`: small standard-library Go review script; no module needed.
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
coherent, and describe the purpose. Push only when requested. No release or
deployment commands are defined.

## Validation

For scout changes, run its own tests:

```sh
go test scripts/scout/scout.go scripts/scout/scout_test.go
```

For documentation and setup changes, check links, installed copies, symlinks,
ignore rules, and `git diff --check`. For future code, add relevant validation
commands here when the implementation provides them.

The scout prints questions for a reviewer; it does not verify the product or
run agents. See [its design](../scripts/scout/DESIGN.md). While editing the scout,
follow [its instructions](../scripts/scout/AGENTS.md).

For inventory changes, follow [its collection rules](../scripts/inventory/README.md)
and run:

```sh
CGO_ENABLED=0 go test ./scripts/inventory
go vet ./scripts/inventory
CGO_ENABLED=0 go build -o /tmp/talkingduck-inventory ./scripts/inventory
```
