# Built-in Markdown preview

Choose `(Built-in viewer)` under Settings → Applications → Markdown viewer.
Click a spec or press `s` in the dashboard to open it. All files share one
preview window, brought forward on opening. The title is `<file name> — <folder label>`, using the deepest containing
folder visible in the dashboard. Other files show their full parent directory. Closing hides the window and focuses the dashboard.
Position and size restore independently of the other windows.

The build registers TalkingDuck as a Markdown viewer for Finder's Open With
menu (`.md`, `.markdown`, `.mdown`), without changing the default association.
The standalone TUI cannot create the built-in desktop window.

| Key | Action |
| --- | --- |
| Esc | Close preview, return to dashboard |
| Tab | Switch to dashboard; Tab there returns to the last Markdown preview |
| Space | Next page |
| Up / Down | Previous / next ToC heading |
| Left / Right | Back / forward through opened files |
| r | Refresh now |
| c | Show / hide ToC |
| h | Search recent files; arrows select, Enter opens |
| Shift + typing | Search the ToC from the top, ignoring case |
| Shift-Up / Shift-Down | Previous / next matching heading, wrapping at either end |
| Other Shift while holding Shift | Next match; release and tap again to repeat |
| Shift-Esc | Clear search and restore the pre-search position |

Hold either Shift key to search. While held, the bottom bar shows
`Type to search: <query>`, a gap, then `↑↓ Matches`, `Esc Reset`, and `⌫ Delete`. Each edit selects the first match and highlights
matching characters in the ToC. A failed or cleared search restores both the
document and ToC scroll offsets from before the search. Releasing Shift keeps a
match; releasing with no match keeps the original position. Shift-Space enters a literal space in the search, including as its first character.

Back/forward and recent-file selection also select the corresponding dashboard
row in keyboard mode, without bringing the dashboard forward. Files absent from
the dashboard leave its selection unchanged. Tab switches window focus without
closing the preview; text fields and open choosers retain normal Tab navigation.

Settings → Keyboard Shortcuts has a separate Markdown preview hotkey, initially
unassigned. It opens the last file, or the recent-file chooser when none is open.
The 50 most recent files retain scroll positions across launches. State lives in
`<settings-file>.markdown-history.json`, separate from portable settings.

The ToC follows the app's tasteful Norton style adapted to dark backgrounds:
compact monospace rows, a shaded panel and edge-to-edge selection. Heading jumps
leave 28 px above the heading and reveal the selected ToC row.

## Rendering and refresh

`markdownview` uses pure-Go Goldmark for GFM and footnotes, YAML for frontmatter,
and Bluemonday for HTML sanitization. Frontmatter mappings appear as a table;
invalid frontmatter remains visible with a warning. GFM checkboxes and Zoo
`[ ]`, `[>]`, `[x]` states render in lists and headings; code remains literal.
Scripts and event handlers are removed. Checkboxes are read-only.

Go reads relative, absolute and `file:` images and embeds them as data URLs,
avoiding WebKit file-access restrictions. Remote HTTP(S) images remain remote.
Limits: 8 MiB document, 20 MiB per local image, 40 MiB total embedded image bytes.
Missing images produce warnings. Markdown links reuse the preview; web/mail links
use the default application. Fenced code scrolls horizontally.

The visible file is polled every 100 ms. Size, modification time and inode detect
ordinary and atomic saves. Rendering waits for 500 ms idle, at most 2 seconds of
continuous changes, with at least 2 seconds between automatic render completions.
Manual refresh and opening are immediate. Rendering runs off the native UI thread.
An unchanged HTML result keeps the DOM; changed content restores the heading and
offset, including after image/font layout. User input cancels pending restoration.

GitHub dark CSS is vendored with its MIT license. Default body font is GitHub's
system stack (San Francisco on macOS), 16 px. Settings enumerate installed font
families, including installed iA Writer Duo/Mono/Quattro variants, and allow 10–40 px.

## Research and checks

Implementation research examined [MarkView](https://github.com/paulhkang94/markview)
and [Ghostwriter](https://github.com/wereturtle/ghostwriter) histories. Relevant
failures included atomic-save watcher replacement, stale ToC nodes after rerender,
old image callbacks changing new pages, unchanged-content scroll resets, local
image permissions and long render work blocking the UI. The design above addresses
these mechanisms without a native Markdown parser or a watcher bound to one inode.

Go tests cover rendering, frontmatter, unsafe HTML, images, task states, heading
IDs, history/positions, refresh timing and key actions. Desktop tests cover atomic
save detection and hotkey routing. Native checks use the installed main app and
real current specs; synthetic fixtures cover repeatable file mutations.
