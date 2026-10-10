# TalkingDuck

See what needs your attention across Codex, Claude, and Grok work folders.
A macOS menu-bar app, dashboard, Markdown viewer, terminal UI, and local MCP server,
built in Go.

## Work dashboard

- Track Git checkouts and linked worktrees, with multiple Zoo tasks per folder.
- See agent activity, tickets, workflow stages, review rounds, subtask progress,
  questions, and suggestions. Unknown or conflicting state stays explicit.
- See changed files and commits ahead/behind. Markdown-only changes stand out.
- Open agent chats, tickets, specs, folders, editors, and Git clients. Copy IDs,
  titles, URLs, and paths. Start a new Codex, Claude, or Grok task.
- Archive rows without changing files; restore them from Settings.
- Use mouse or keyboard in a compact, dark Norton-style layout.
- Keep status visible in the menu bar, with standard/compact layouts and
  per-folder labels. Toggle the dashboard with Ctrl-T or a custom hotkey.
- Refresh automatically from filesystem events, with periodic recovery and
  background Git fetch every 15 minutes. Manual refresh is always available.

Supports local Codex/ChatGPT and Claude desktop sessions, plus Grok Build in cmux.
See [collection behavior and limits](cmd/talkingduck/README.md).

## Markdown viewer

Choose **Settings → Applications → Markdown viewer → (Built-in viewer)**.
Press `s` on a dashboard row, click its spec, or use Finder's Open With menu.

- GitHub dark styling; GFM tables, links, footnotes, and task lists, rendered in
  pure Go. Zoo `[ ]`, `[>]`, and `[x]` markers work in lists and headings.
- YAML frontmatter shown as a table. Inline local and remote images.
- Compact side ToC with keyboard navigation and highlighted search matches.
- Hold either Shift and type to search headings from the top. Matching ignores
  case and spaces: `UICH` finds `UI Changes`. Release Shift to keep a match;
  a failed search restores the original position.
- Smooth page scrolling. Live reload with a 0.5-second cooldown and 2-second
  throttle, preserving the reading position across edits and atomic saves.
- Searchable history of 50 recent files, with scroll positions saved across launches.
- One reused window, remembered position and size, and a separate global hotkey.
  Tab switches between viewer and dashboard; history navigation selects the matching dashboard row.
- Configurable font family and size, including installed iA Writer fonts.
  Defaults to GitHub's system font stack at 16 px.

| Key | Action |
| --- | --- |
| Esc / Tab | Close / switch to dashboard |
| Space / Shift-Space | Next / previous page; Shift-Space adds a space once searching |
| ↑ / ↓ | Previous / next ToC heading |
| ← / → | File history |
| h | Search recent files |
| r / c | Refresh / toggle ToC |
| Hold Shift + type | Search ToC |
| Shift-↑ / Shift-↓ | Previous / next match |
| Tap the other Shift | Next match while holding Shift |
| Shift-Esc | Clear search and restore original position |

Markdown links open in the same viewer; web links use the default browser.
See [viewer details](docs/markdown-viewer.md).

## Build and run

Requires Go (see [go.mod](go.mod)) and Xcode Command Line Tools.

```sh
./desktop/build.sh
open "$HOME/Applications/TalkingDuck.app"
```

The build installs an ad-hoc-signed app into `~/Applications`. Quit the running
app before rebuilding, then launch again. No notarization or login-item setup
is included.

Add folders in Settings (`⌘,` or Ctrl-S). Discovered folders start hidden;
select Show to monitor them. Menu-bar labels are configured separately.
Settings also choose editors, Git clients, Markdown viewers, ticket trackers,
and global hotkeys. Custom app commands and ticket URL templates are supported.
Window positions and sizes are remembered.

GUI, TUI, CLI, and MCP share `~/.config/zoo/talkingduck-settings.json`, honoring
`XDG_CONFIG_HOME`. See [folder settings](docs/settings.md),
[app and ticket launchers](docs/launchers.md), and [desktop details](desktop/README.md).

## Dashboard and terminal keys

Uppercase keys mean Shift; copy chords are typed in sequence.

| Key | Action |
| --- | --- |
| ↑ / ↓ | Select row |
| a / t / s | Open agent chat / ticket / spec |
| S | Reveal spec in Finder |
| f / e / g | Open folder / editor / Git client |
| A, then x / c / g | New Codex / Claude / Grok task |
| ct / cT | Copy ticket ID / URL |
| cs / cS / cf | Copy spec title / spec path / folder path |
| r / R | Refresh selected folder / all folders |
| X | Archive row |
| Tab | Back to viewer (desktop) |
| Esc | Cancel chooser or chord; clear dashboard keyboard selection |

Mouse users can click links and copy controls. Multiple matches open a chooser.
Titles wrap; keyboard selection stays visible during scrolling and refresh.

## CLI

```sh
go run ./cmd/talkingduck          # terminal UI
go run ./cmd/talkingduck -status  # one-shot table
go run ./cmd/talkingduck -ai      # compact text for agents
go run ./cmd/talkingduck -gui     # installed Mac app
```

Pass folder paths to override the configured scope. Shift-Q or Ctrl-C quits the
TUI. The standalone TUI uses external Markdown viewers; it cannot create the
built-in preview window.

Build the CLI without cgo:

```sh
CGO_ENABLED=0 go build -o talkingduck ./cmd/talkingduck
```

## MCP

While the Mac app runs, local agents can connect to
`http://127.0.0.1:42389/mcp`, even with the dashboard hidden. Tools read the shared
inventory, refresh a folder, manage monitored folders, and set menu-bar labels.
The port is configurable; Settings shows the active address.
See [MCP setup and tools](mcpserver/README.md).

## Development

- [Documentation map](docs/README.md)
- [Development and validation](docs/development.md)
- [Product vision](docs/product.md)
- [Agent instructions](AGENTS.md)

Zoo skills live in `.agents/skills`, with a `.claude/skills` link. Project rules
live in `.zoo/`; specs in `.spec/`. The repository includes Codex/Claude review
roles and a [review scout](cmd/scout/DESIGN.md).
