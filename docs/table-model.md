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
- Chord keys are strings (`a`, `t`, `s`, `f`). Multiple matching items remain
  separate choices. Hidden repeated folder labels retain actions for their row.
- Open actions carry a typed target, never a shell command. Missing links have
  no open action; future Git actions can use `g` without inventing a target today.

`table.Format` computes Unicode cell widths, alignment, multiline height, and
flexible-column wrapping while retaining colored spans and source row indices.
Narrow screens clip or scroll fixed columns; data is not silently discarded.
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
