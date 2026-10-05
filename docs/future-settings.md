# Future settings

Remaining configuration backlog. [Folder settings](settings.md) are implemented
in the shared XDG settings file. Settings also shows the live MCP server address
or startup error. Open it from the overview, tray menu, or ⌘,.

| Setting | Current behavior | Future control |
| --- | --- | --- |
| Folder order | Saved insertion order; discovered folders appended hidden | Reordering |
| Linear workspace | `bubblehouse`; all ticket links use `https://linear.app/bubblehouse/issue/<ID>` | Workspace, with per-project overrides if needed; consider other trackers later |
| Markdown preview | Spec titles open in Marked | Installed-app picker, system default, or built-in preview |
| Agent chat destination | Native Codex and Claude Code deep links when IDs are known | Per-agent app preference and future remote/cloud routing; see [chat links](agent-chat-links.md) |
| Ticket destination | macOS opens the HTTPS URL with its default handler | Optional browser/Linear-app preference |
| Remote fetch | Startup and every 15 minutes independently of scanning; failed attempts also throttled | Fetch interval and enable/disable |
| Refresh scheduling | Cheap cooldown 1–5s/throttle 5s; expensive cooldown 5–20s/throttle 60s; process checks 30s; recovery 10min | Per-queue timing; optional pause/resume |
| Menu bar density | Standard by default; Compact toggle lasts for the current run | Persist selected density |
| Folder summary selection | First listed spec determines the tray stage/activity | Decide how multiple specs and attention states should be prioritized |
| Agent collection | All local sessions; `--no-agents` and `--agent-limit` supported | Provider selection and collection limits |
| MCP | Enabled while GUI runs; port 42389, overridden by `--mcp-port` | Enable/disable and port; keep loopback-only binding |
| Startup | Manual launch, no login registration | Launch at login; initial window visibility |

Implementation pointers: `desktop/main.go` (startup, tray, refresh interval),
`desktop/ticket_link.go` (Linear workspace), `desktop/spec_preview.go` (Marked),
`cli/cli.go` (collection flags), `settings/settings.go` (folder settings),
`mcpserver/server.go` (MCP port and loopback transport).

Do not make non-loopback listening a preference by accident. Remote access would
need a separate design for authentication and inventory privacy.
