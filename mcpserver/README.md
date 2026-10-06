# Local inventory MCP

The desktop app serves Streamable HTTP at `http://127.0.0.1:42389/mcp` while
running, including when its window is hidden. It shares the same in-memory cache
as the window and menu bar. The standalone CLI does not start a server.

Tools:

- `list_zoo_tasks {}` returns one MCP text block using the exact CLI `-ai`
  renderer, followed by compact per-folder cache state and observation times.
  Reads are immediate and do not trigger a scan.
- `refresh_folder {"folder":"back2"}` accepts a configured absolute checkout
  path or unique basename. It waits for the current scan, refreshes only that
  checkout, publishes to the window/tray, and returns the updated compact
  inventory. Unknown or ambiguous names return a tool error. Failed refreshes
  retain earlier cached data; `list_zoo_tasks` reports the error and original age.

Connect any Streamable HTTP MCP client to the URL above. For clients using an
`mcpServers` JSON configuration:

```json
{"mcpServers":{"talkingduck":{"url":"http://127.0.0.1:42389/mcp"}}}
```

`--mcp-port=42390` changes the port when launching the desktop executable with
`--gui`. The address is always IPv4 loopback; no flag enables external binding.
Browser Origin headers and non-loopback Host headers are rejected. No auth token
is required: other local processes can read inventory and request configured
folder scans. No CORS access is enabled.

The app closes the listener and HTTP connections on quit. The collector cannot
cancel an already-running folder scan midway; its own command/database timeouts
still apply. A port conflict is logged and displayed; the GUI keeps running.

Uses the [official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk),
pinned in `go.mod`. HTTP integration tests cover initialization, discovery, text
results, targeted refresh, invalid folders, and Host/Origin checks:

```sh
CGO_ENABLED=0 go test ./mcpserver ./cli
go test -race ./mcpserver ./desktop
```

Folder settings tools (same settings as GUI/TUI):

- `list_folders {}` returns all paths, including hidden/discovered entries, with
  Show/Hide, menu bar shortcut, and current cache status. Non-Git and missing paths are excluded.
- `add_monitored_folder {"folder":"/absolute/path"}` saves Show and starts monitoring.
- `remove_monitored_folder {"folder":"/absolute/path"}` saves Hide; deletes no files
  and retains the entry so rediscovery cannot re-enable it.
- `update_folder_menubar_item_shortcut {"folder":"/absolute/path","shortcut":"a"}`
  sets the string label; `""` means Hide. Allowed values: 0–19, A–Z, a–z.

See [folder settings](../docs/settings.md) for persistence and live reconfiguration.

Adding a monitored folder requires a Git checkout root (ordinary repository,
linked worktree, or submodule), matching Settings validation.
