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
Headers use the same alignment as values. Q displays zero as a dim `-`;
other numeric columns retain zero.

Copy controls touch item text with no added space. A cell line with one visible
item extends hover across its full width, including whitespace; blank space
still only selects the row. Lines with multiple items keep individual hover targets.

GUI and TUI share column order: Folder, Agent, Ticket, Stage, Uber, Run, Ask,
Rev, Tasks, Q, Mod, New, .md, Behind, Ahead, Title. Uber/Run/Ask labels stay blank.

Keyboard: `cT` copies the ticket URL; `ct` copies its number. Multiple tickets
use the same chooser. GUI also accepts `CT`.


The GUI uses a DOM-independent controller for mouse/keyboard selection and
actions. Hover and keyboard row identities remain separate across snapshots.
Esc hides keyboard selection; the bottom action bar targets the effective row.
Action status is short and right-aligned. See desktop/README.md for keys and tests.
