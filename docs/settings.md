# Settings

GUI, TUI, one-shot CLI, and MCP share `$XDG_CONFIG_HOME/zoo/talkingduck-settings.json`.
When `XDG_CONFIG_HOME` is unset, use `~/.config/zoo/talkingduck-settings.json`.
`-settings-file PATH` overrides it. No built-in worktrees and no automatic legacy
folder migration: a new installation starts empty. This installation's previous
ten folders were explicitly saved during setup.

Settings has one list, with:

- **Use:** checkbox before the path. Only checked folders appear in inventory and receive
  Git/spec/session scans, process probes, or fetches. Unchecking retains the entry.
- **Menu Bar Shortcut:** Hide, 0–19, A–Z, a–z. Stored as a string; empty excludes
  the folder from the menu bar. A nonempty value includes it only while monitored.
  Case is distinct; duplicate labels are allowed. Two-digit labels retain narrow
  digit-1 rendering. With all items hidden, a neutral app icon keeps Settings reachable.
- **Add Folder…:** native folder picker, adds or re-enables a Git checkout root.
  New folders default to menu bar Hide.

While Settings is visible, the GUI discovers folder paths from Codex databases,
Claude metadata, and Grok metadata, then repeats every minute. It also reads Git's
worktree registry for configured and discovered checkouts, including worktrees
outside the main checkout's directory and those without an agent session yet.
This reads no transcripts and runs no Git commands. Missing/stale worktrees are
ignored. A new linked worktree inherits its main checkout’s Use setting. Other
discovered folders start unchecked. Git-registry discovery runs every minute while
displays are awake, even with Settings closed; provider metadata scans still
require Settings to be visible.

A newly discovered linked worktree gets a menu bar shortcut when its main checkout
has one and its assigned agent index is 1–19. A literal `AGENT_INDEX` assignment in
that worktree's `.env.local` takes precedence over a numeric folder-name suffix.
Optional `export`, single/double quotes, and trailing comments are supported;
shell expressions are never evaluated. The implicit default of 1 does not qualify;
an explicit `AGENT_INDEX=1` does. Invalid assignments and out-of-range indexes get
no shortcut. Existing shortcuts and explicit Hide choices are never overwritten.
Monitoring and shortcut inheritance happen only on first discovery; later changes
to the main checkout do not overwrite the worktree’s own choices.
Previously hidden entries stay hidden. Linked-worktree entries store `worktree_of`
so a later discovery pass can silently remove them when their directories no
longer exist. Existing live entries gain this metadata on discovery. Permission
errors do not trigger removal, and missing ordinary checkouts stay configured.
Only Git checkout roots are listed or monitored. Ordinary
repositories, linked worktrees, and submodules qualify; non-repositories, nested
working directories, and missing paths do not. Existing excluded entries remain
in the settings file. Validation reads Git metadata without launching Git.

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
- `add_monitored_folder {"folder":"/absolute/project"}`: Show; validate Git checkout root.
- `remove_monitored_folder {"folder":"/absolute/project"}`: Hide; keep settings.
- `update_folder_menubar_item_shortcut {"folder":"/absolute/project","shortcut":"a"}`:
  update label; use `""` for Hide. Does not change monitoring visibility.

These operate on the GUI's shared settings and update its window/tray immediately.

## Applications and tickets

The same file stores `applications.editor`, `applications.git_client`, `applications.markdown`, and
`applications.tracker`. Editor/Git selections contain `id`, plus `path` and `args`
for Custom. Defaults are Zed and Sublime Merge. Tracker selections contain `id`,
the pasted `url`, derived `account_slug`, and a `custom` profile when applicable.
No workspace account is built in; this installation's existing Linear workspace
was explicitly saved. Other installations must supply their own tracker URL.

App and tracker forms autosave after 300 ms idle, at most 1.5 s of continuous
editing, with at least 500 ms between save completion and the next save. Folder
checkboxes and shortcut choices save immediately. No Apply button or success message;
invalid drafts preserve the working configuration. Open/copy actions reload the
chosen settings file, including `-settings-file`, so GUI and TUI stay in sync.
See [launchers](launchers.md) for schemas, placeholders, and platform limits.
The current MCP tools configure folders only.

Settings uses grouped forms and one scrolling document, with aligned labels and
controls. See [macOS UI conventions](macos-settings-ui.md).

Connection includes a copy button for the MCP address. Configuration File shows
JSON format and the active settings path; Reveal selects that file in Finder.

Markdown viewer defaults to `(Default)` (OS association). `(Text editor)` follows
Editor. `(Built-in viewer)` opens the shared desktop preview; see [Markdown preview](markdown-viewer.md). Other viewers and Custom open the spec file directly. App dropdowns are
alphabetical, platform-filtered, and mark undetected installations with muted
text. Quick checks are cached until restart; Custom handles undetected locations.

## Recently archived

Shows up to 20 archived specs or task-free rows, newest first. Each compact row
shows the spec filename, a muted relative date when known, and Unarchive. Hover
the name for the full worktree/spec path. Legacy entries have no date and sort last.
Unarchive removes the display tombstone and refreshes monitored worktrees; it
does not enable hidden folders or modify repository files. Older archives remain
stored even when outside this list.

## Global hotkeys

Settings → Keyboard Shortcuts has three inline recorders: Global hotkey,
Additional hotkey, and Markdown preview. Click to record; press a combination; Escape cancels. The ×
button clears a binding. Changes save immediately and apply without relaunching.
The original Control-T default remains until configured; clearing both disables
the dashboard bindings. The first two shortcuts toggle the dashboard; the third
opens the last Markdown preview.

`global_hotkeys` stores three accelerator strings in the shared settings JSON.
An absent field means `["Ctrl+T", "", ""]`; `["", "", ""]` means disabled. Legacy two-entry arrays
leave the Markdown shortcut unassigned. Example:
`["Ctrl+Alt+Shift+Super+T", "F18", "Ctrl+Alt+M"]`. `Super` means Command on macOS.

The macOS backend supports any combination of Control, Option, Shift, and Command,
letters, digits, punctuation, navigation keys, and F1–F20. Fn/Globe, modifier-only
bindings, and media keys are unsupported. Physical key positions follow the
native backend’s ANSI/QWERTY map. Standard system/menu shortcuts can be intercepted
before the recorder receives them. Another app’s occupied shortcut is reported
as an error; rejected edits preserve the previous setting.

Global bindings pause while recording and resume on completion, cancellation, or
Settings losing focus. The recorder is local HTML/JavaScript; native registration
uses the existing Wails Carbon backend without Accessibility permission.

Built-in Markdown Preview settings store `markdown_appearance.font` (default
`github`) and `size` (default 16 px). Font choices come from installed macOS
families. These fields use the same autosave behavior as Applications.

## Worktree display names

Linked worktrees drop trailing path components equal to the main checkout's name:
`~/.codex/worktrees/c9e4/backend` under main checkout `backend` displays as `c9e4`.
Ordinary checkout names and descriptive worktree names remain unchanged. Full
paths still identify rows and drive open/copy actions; Markdown titles reuse the
same display label.

The rule also covers custom roots and repeated repository-name wrappers without
hardcoding user home paths. Claude's [default layout](https://code.claude.com/docs/en/worktrees)
is `<repo>/.claude/worktrees/<name>`; Grok's [documented examples](https://github.com/xai-org/grok-build/blob/main/crates/codegen/xai-grok-pager/docs/user-guide/17-sessions.md)
use `~/.grok/worktrees/<repo>/<worktree-name>`. Both retain the distinguishing name.
