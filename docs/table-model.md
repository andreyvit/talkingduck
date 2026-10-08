# Shared table model

`table.Table` is UI-independent, serializable presentation data. `cli.BuildTable`
projects inventory into it; desktop and terminal renderers consume it.
Legacy string rows remain for AI output and compatibility with existing callers.

- Columns: stable ID, label, alignment, minimum width, flexible width, optional
  terminal counter prefix. Title takes remaining space; cells are top aligned.
- Rows: stable worktree/spec ID, full worktree path for refresh/grouping, semantic
  state (`running`, `attention`, `pushed`, `error`), cells in column order.
- Cells: vertical lines containing inline items.
- Items: text, semantic color, bold, tooltip, optional open action and independent
  copy text. Copying a ticket copies its number, not a URL. Spec copy preserves
  the existing title-copy behavior. Folder copy uses its full path.
- Action keys are strings (`a`, `t`, `s`, `f`). Multiple matching items remain
  separate choices. Open keys are direct; only copy uses a `c` prefix.
  Hidden repeated folder labels retain actions for their row.
- Open actions carry a typed target, never a shell command. Missing links have
  no open action; future Git actions can use `g` without inventing a target today.

`table.Format` computes Unicode cell widths, alignment, multiline height, and
flexible-column wrapping while retaining colored spans and source row indices.
The TUI wraps titles and clips fixed columns on narrow screens; no horizontal scrolling.
Historical row color overrides item colors. Terminal and browser palettes map
semantic names to colors suitable for their backgrounds.

Tests cover Unicode alignment, multiline cells, wrapping, inline items, hidden
folder actions, ticket copy values, and pushed-row dimming.

Interactive terminal layout uses `FormatWithOptions` to reserve copy controls.
Spans retain source item references and control roles through wrapping and
alignment; `HitTest` maps cell coordinates back to them. Hover only changes
visibility, never column widths. Plain output reserves no controls.

The TUI translates viewport coordinates into layout coordinates. Clicks activate
on release over the pressed target; inventory updates and resizes cancel pending
clicks. Choosers own mouse input while open. Wheel scrolling is independent of
selection; arrow-key selection reveals the selected row again.

Before the first scan completes, a worktree shows its folder and `SCANNING` in
Stage; other cells stay empty. Pending collection is not an error. Failed scans
still display their error.

TUI headers use colored bold text on the normal background; numeric labels use
their column colors. Uber, progress, and input have blank labels and one-cell minimums.
Headers use the same alignment as values. Q/S displays zero as a dim `-`;
other numeric columns retain zero.

Copy controls touch item text with no added space. A cell line with one visible
item extends hover across its full width, including whitespace; blank space
still only selects the row. Lines with multiple items keep individual hover targets.

GUI and TUI share column order: Agent, Folder, Ticket, Stage, Uber, Run, Ask,
Rev, Tasks, Q/S, Mod, Behind, Ahead, Title. Uber/Run/Ask labels stay blank.
Mod shows `0` for no changes, blue `.md` for Markdown-only changes anywhere in
the worktree (case-insensitive `.md`), or a count of all changed paths when any
non-Markdown file changed. Each path counts once across staged/unstaged changes.
Unknown components keep the total unknown. Raw inventory and AI output retain
three disjoint counts: tracked non-Markdown, untracked non-Markdown, all Markdown.
Q/S is right-aligned and orange, distinct from yellow Mod. Known positive totals
use a filled badge (rounded in GUI, inverted in TUI); zero stays a dim `-`.
`?` means questions or suggestions could not be counted, not an open question.

Absent agents display blank; unknown agents retain `?`.
GUI agent cells use 26-point vector icons with a margin around the artwork matching the menu-bar motifs, with
agent names in accessible labels and tooltips. Set `USE_AGENT_ICONS` to false in
`desktop/frontend/agent-icons.mjs` to restore names. Terminal output keeps names.
The GUI targets an 864-point window (half a 1728-point MacBook display): below
900 points, column gaps tighten; titles wrap without reducing the text size.

Keyboard: `cT` copies the ticket URL; `ct` copies its number. Multiple tickets
use the same chooser. GUI also accepts `CT`. `cS` copies the full spec path in both UIs; `cs`
continues to copy its title.


The GUI uses a DOM-independent controller for mouse/keyboard selection and
actions. Hover and keyboard row identities remain separate across snapshots.
Esc hides keyboard selection; the bottom action bar targets the effective row.
Action status is short and right-aligned. See desktop/README.md for keys and tests.

The GUI shows one plain `↑↓ Select` hint; selection arrows are not clickable.

Rows carry an archive action whose target is the relative spec path (empty for
the task-free row). Shift-X uses that action on the effective row in either UI.
Archive state filters shared inventory, including MCP output, without changing
Git status or the spec. The footer exposes X Archive.

Rows also carry `actions`: items without a visible cell. Editor (`e`) and Git
client (`g`) actions target the row's worktree, including grouped/historical rows.
Both clients resolve open-ticket and copy-ticket-URL through current shared
settings. The table never embeds a personal tracker workspace URL.

Shift-S reveals the effective row's spec in Finder (Explorer on Windows; the
containing folder on Linux). The GUI swaps `s Spec` for `S Reveal spec` while
Shift is held and restores it on key release or window blur. The TUI supports
Shift-S without a dynamic modifier hint; ordinary terminals lack Shift-up events.

Executing a row action (including a direct link or copy button) selects that row
in keyboard mode before handing off focus. Returning via Ctrl-T retains it.
Pointer movement over the selected row leaves keyboard mode intact; movement
over another row switches to mouse mode only while the document is focused.
Window activation follows the row currently under the pointer, without requiring
a click; outside the table or over the keyboard-selected row, selection stays.
A row pointer-down also explicitly enters mouse mode.
Inactive hover can update the remembered mouse row without changing the keyboard
selection or mode. Choosers pin their originating row when a choice executes.
