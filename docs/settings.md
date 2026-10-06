# Settings

GUI, TUI, one-shot CLI, and MCP share `$XDG_CONFIG_HOME/zoo/talkingduck-settings.json`.
When `XDG_CONFIG_HOME` is unset, use `~/.config/zoo/talkingduck-settings.json`.
`-settings-file PATH` overrides it. No built-in worktrees and no automatic legacy
folder migration: a new installation starts empty. This installation's previous
ten folders were explicitly saved during setup.

Settings has one list, with:

- **Monitoring:** Show or Hide. Only Show folders appear in inventory and receive
  Git/spec/session scans, process probes, or fetches. Hide retains the entry.
- **Menu Bar Shortcut:** Hide, 0–19, A–Z, a–z. Stored as a string; empty excludes
  the folder from the menu bar. A nonempty value includes it only while monitored.
  Case is distinct; duplicate labels are allowed. Two-digit labels retain narrow
  digit-1 rendering. With all items hidden, a neutral app icon keeps Settings reachable.
- **Add Folder…:** native folder picker, adds or re-enables the folder as Show.
  New folders default to menu bar Hide.

The GUI discovers existing folder paths from Codex databases, Claude metadata,
and Grok metadata on startup and every minute. This reads no transcripts and runs
no Git commands. Discovered paths join the same list as Hide. Previously hidden
entries stay hidden. Missing saved folders remain listed with a missing status.

Changes apply live. Inventory retains cached observations and scan locks when
membership changes. Event sources restart for the new set; queue throttles remain.
Queued work for hidden folders is skipped; a scan already underway may finish but
cannot restore a hidden row. GUI and TUI check the small settings file every second,
independently of inventory work. Explicit positional paths override the list for
a CLI/TUI run. A GUI launched with positional paths keeps that temporary scope
until a folder setting is changed in that app or through its MCP server.

Writes use an advisory lock and atomic replacement, preserving other processes'
changes. Invalid settings produce an error rather than being replaced. Task history
and archived rows remain in the separate `state.json`; they are not folder settings.

Example:

```json
{
  "version": 1,
  "folders": [
    {"path": "/absolute/project", "visible": true, "menubar_item_shortcut": "1"},
    {"path": "/absolute/other", "visible": false, "menubar_item_shortcut": "", "discovered": true}
  ]
}
```

MCP actions:

- `list_folders {}`: saved paths, Show/Hide, shortcut, discovery source, and status.
- `add_monitored_folder {"folder":"/absolute/project"}`: Show; validate directory.
- `remove_monitored_folder {"folder":"/absolute/project"}`: Hide; keep settings.
- `update_folder_menubar_item_shortcut {"folder":"/absolute/project","shortcut":"a"}`:
  update label; use `""` for Hide. Does not change monitoring visibility.

These operate on the GUI's shared settings and update its window/tray immediately.

## Applications and tickets

The same file stores `applications.editor`, `applications.git_client`, and
`applications.tracker`. Editor/Git selections contain `id`, plus `path` and `args`
for Custom. Defaults are Zed and Sublime Merge. Tracker selections contain `id`,
the pasted `url`, derived `account_slug`, and a `custom` profile when applicable.
No workspace account is built in; this installation's existing Linear workspace
was explicitly saved. Other installations must supply their own tracker URL.

Simple app choices save immediately. Custom commands and tracker forms use Apply;
invalid drafts preserve the working configuration. Open/copy actions reload the
chosen settings file, including `-settings-file`, so GUI and TUI stay in sync.
See [launchers](launchers.md) for schemas, placeholders, and platform limits.
The current MCP tools configure folders only.

Settings uses grouped forms and one scrolling document, with aligned labels and
controls. See [macOS UI conventions](macos-settings-ui.md).
