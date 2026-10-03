# AI documentation

Start here; read only what the task needs.

- [product.md](product.md): purpose, user needs, boundaries, open decisions.
- [Agent chat links](agent-chat-links.md): Codex/Claude URL routes, session IDs, and limits.
- [Future settings](future-settings.md): hardcoded defaults and planned configuration controls.
- [development.md](development.md): repository layout, work conventions, validation.
- [Desktop prototype](../desktop/README.md): cached inventory window and Wails menu bar, standard/compact layouts, native width shim, build and checks.
- [Inventory behavior spec](../.spec/20261002-worktree-task-inventory.md): worktree/task/session hierarchy, compact table, stages, correlation, and limits.
- [Local work inventory](../cmd/talkingduck/README.md): one-shot Git/spec collector, ordering, and limits.
- [Live refresh](live-refresh.md): five-second cheap updates, background scans, signals, and synchronization.
- [Shared inventory builder](../inventory/README.md): collection, per-worktree refresh, and cached snapshots for CLI/desktop/MCP callers.
- [Local MCP server](../mcpserver/README.md): loopback HTTP tools for cached inventory and targeted refresh.
- [Shared CLI](../cli/README.md): flags, output, and entry-point integration.
- [Agent collection research](agent-collection-research.md): local schemas, live checks, monitor comparisons, optional hooks.
- [Agent inventory instructions](agent-inventory-instructions.md): collect once, then enrich unknowns with app tools or UI.
- [Scout design](../cmd/scout/DESIGN.md): review-tool behavior and extension rules.

Keep durable project knowledge here. Link new topic docs from this index.
Keep `AGENTS.md` short: universal rules and pointers. Zoo workflow configuration
belongs in `.zoo/`; task plans in `.spec/`; proposals in `.proposals/`.
Temporary research and evidence belong in ignored `.tasks/`.

Update facts when behavior changes. Record sources and uncertainty for discoveries.
Avoid duplicating instructions or turning tentative ideas into settled architecture.
