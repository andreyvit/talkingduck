# Menu bar app and inventory window

macOS-only Wails v3.0.0-beta.27 prototype. The main window uses live
`inventory.Inventory` cache and the CLI's shared `BuildTable` rows. The menu bar
uses the same cached checkout states and the shared `menubarimage` renderer. No frontend
dependencies, Node, generated bindings, or external web server.

From the repository root:

```sh
./desktop/build.sh
open "$HOME/Applications/TalkingDuck.app"
```

Requires Xcode Command Line Tools. The build script compiles with cgo, targets
macOS 13+, packages the app, and signs it ad hoc for local use. It does not
notarize or register autostart. It installs into `~/Applications/TalkingDuck.app`;
temporary build output under `.tasks/desktop` is removed when the script exits.
Quit the app before rebuilding, then launch the installed bundle with `open` as
above, or through Finder. Do not launch the GUI with `go run` or execute its
binary directly from the agent shell.

The desktop executable defaults to GUI mode, including direct execution. Both
binaries share `-gui`, `-tui`, `-status`, and `-ai`; the command-line binary defaults
to TUI. Explicit modes override the default and cannot be combined. Use
`~/Applications/TalkingDuck.app/Contents/MacOS/talkingduck-desktop -tui` to open its TUI in a terminal. The standalone CLI remains cgo-free. The GUI creates one in-memory cache and
updates folders through independent cheap and expensive queues, while hidden, but pause when displays sleep. Covered windows defer rendering; see [energy behavior](../docs/energy.md).
Native filesystem events, changed process statuses (checked every 30 seconds),
and ten-minute recovery feed those queues. Cheap cooldown is 1–5 seconds with a
5-second completion throttle; expensive cooldown is 5–20 seconds with a 60-second
completion throttle. Manual/MCP refreshes join the expensive queue. See
[event-driven refresh](../docs/live-refresh.md). Collection flags and paths apply.
Previous rows remain visible during refresh; failed folders retain their last
successful data and show a warning. Unscanned folders show explicit placeholders.
Refreshing rows show a small activity overlay in the first column, over the agent
icon when present, without changing layout. The bottom bar contains only keyboard hints. Notices
appear in a separate full-width bar above the limits panel and bottom controls.
Action notices clear after four seconds; connection status, errors and warnings
remain until cleared by their source. The message bar disappears when empty. Quit stops scheduling; an in-flight collector
call has its own timeouts and cannot be cancelled midway.
`--gui=false` overrides the bundle environment when needed.

- Left-click the tray image: hide the overview if visible and focused; otherwise
  show and focus it.
- Right-click: native Wails-created menu with open, compact mode, refresh, and quit.
  A local AppKit event monitor consumes right-down on the status button and opens
  that menu directly with `popUpMenuPositioningItem`. It leaves left-clicks alone.
  The earlier `OnRightClick` → `OpenMenu` workaround still failed on alternate
  clicks: `OpenMenu` also synthesizes a button mouse-down. This adapter avoids
  both that path and temporary `statusItem.menu` assignment.
  Build/vet pass; repeated physical right-clicks still need verification because
  the available UI automation exposes the window but not the status item.
- Standard (default): 123 x 23-point image, 5x7 font, 131-point tray width.
- Compact: 99 x 21-point image, 4x6 font, 105-point tray width.
- Switch layouts live with the native menu checkbox.
  The selection lasts for this run; preferences are not saved yet.
- Click a linked Codex/Claude name to open that primary chat in its desktop app.
  See [agent chat links](../docs/agent-chat-links.md) for supported IDs and limitations.
- Click any ticket reference to open it in the configured tracker with the default URL handler.
  Each ticket on a multiline row is linked separately; links underline on hover.
  Configure the tracker and workspace in Settings; see [settings](../docs/settings.md).
- Click a spec title to open its file with the configured Markdown viewer. Titles underline on hover;
  non-task rows stay plain text. Missing files or app launch errors appear in the window.
- Closing the window hides it.
- The inventory window uses the shared table model and row ordering,
  including multiline tickets and partial errors. Monospace text, whitespace,
  and subtle alternating task-row backgrounds separate columns; no gridlines.
  Rows extend to both window edges. No top heading or toolbar; the bottom bar
  carries clickable key hints and right-aligned action status.
- Tray numbers follow configured checkout order. The first listed task supplies
  the stage, preserving the prototype's selection rule. Explicit questions or
  completed turns need attention; running turns get an outline. Failed/pending
  entries are unknown. Task-free checkouts show `-` when clean and `d` when
  modified (including informational Markdown changes). Multiple-task priority
  remains a future decision.

## Local MCP

Settings (bottom-bar hint, Ctrl-S, tray menu, or ⌘,) shows the actual MCP address,
including a custom port, in a selectable field. Startup failures appear there too.
Closing Settings hides it for reuse.

While the app runs, its shared cache is available at `http://127.0.0.1:42389/mcp`.
`list_zoo_tasks` returns compact CLI `-ai` text; `refresh_folder` updates one
configured checkout and the UI. See [MCP setup and tools](../mcpserver/README.md).

## Variable-width workaround

Wails creates and owns the `NSStatusItem`, native menu, handlers, and lifecycle.
Its stock macOS image setter forces a square. `wide_tray_darwin.go` obtains the
existing native handle using reflection on `SystemTray.impl.nsStatusItem`, then
uses public AppKit APIs to restore the selected logical image size and
status-item width. A 2x bitmap supplies Retina pixels. Template rendering
adapts foreground and transparent text cutouts to the menu bar appearance.

No Wails fork, Go memory offsets, Objective-C method swizzling, or replacement
status item. The Go field access depends on private framework fields: the shim checks
build metadata for the exact unmodified Wails version and checks reflected field
kinds before reading the pointer. The menu adapter also reads `nsMenu`; its
monitor lives for the app process (this prototype does not recreate trays).
Revalidate both adapters before upgrading Wails.

Tray updates decode the PNG, apply colors, assign the image and set its logical
size in one native main-queue operation. Do not use Wails' image setters:
 they first schedule a run-loop callback, which then enqueues another
image assignment. The old adapter's separate resize could run before that
assignment and be overwritten with a square image. Call `updateWideTray` off the
AppKit main thread. Updates are serialized and native checks verify dimensions.
Geometry/visibility changes are logged as `TalkingDuck tray:` to help distinguish
image failures from macOS menu-bar placement or visibility settings.

Always update images through this adapter: calling Wails' setter alone restores
the square. An unknown Wails version fails visibly instead of silently reading
an assumed memory layout. A public native-handle or icon-size API would remove
this dependency.

## Validation

```sh
node --test desktop/tests/*.test.mjs
CGO_ENABLED=0 go test ./menubarimage ./cli
CGO_ENABLED=0 go vet ./cmd/scratch-imagegen ./menubarimage
CGO_ENABLED=0 go build -o .tasks/desktop/talkingduck-cli ./cmd/talkingduck
CGO_ENABLED=1 go test -race ./desktop
CGO_ENABLED=1 go vet ./desktop
./desktop/build.sh
plutil -lint desktop/build/Info.plist
```

Run the app, verify startup geometry in stderr, toggle styles, reopen the overview,
exercise the native menu, and quit. Native menu-bar visibility still depends on
available space (especially around a MacBook notch). The compact renderer uses
a 4x6 font, 19x10 cells, and 1-point gaps.
This does not establish readability on every display or verification on other
macOS versions.

Ticket numbers and linked task titles have small copy buttons. Space is reserved;
buttons appear faintly on text hover or keyboard focus, brighten on button hover,
and briefly show a checkmark after copying the exact text through Wails.

Scratch-only agent backdrop experiment: `go run ./cmd/scratch-imagegen
-agent-backdrops -out .tasks/agent-backdrops`. Repeats Claude/Codex/Grok-inspired
cropped motifs at the right edge. This experiment renders at four pixels per
logical point before the scratch preview scaling; it does not change the app.

The live tray now uses the approved cropped agent motifs (Claude sunburst,
Codex loops, Grok slashed ring), selected from each folder’s first task. Unknown
agents have no motif. Both densities use equal-width cells, left-aligned labels,
and narrow vertical ones only within multi-digit numbers. Motifs render at 4×
backing resolution while AppKit receives the original logical dimensions.

Scratch `*-preview-1x.png` files now use logical-size pixels; `*-preview-4x.png`
uses nearest-neighbor enlargement of that actual-size raster. `*-icon@2x.png`
is a correctly sized Retina asset. Downsampling averages premultiplied coverage;
AppKit may rasterize differently. Agent strokes are wider and vertically centered.

Agent motifs use explicit logical-pixel masks, with clear negative space rather
than overlapping tiny curves. Each stroke survives 1× display rasterization;
standard and compact layouts use 9- and 8-pixel-high masks respectively.

The desktop starts background `git fetch --all --quiet` at startup and every
15 minutes, independently of inventory processing.
At most five fetches run concurrently; remaining folders queue in the background.
New refs trigger filesystem events and enter the expensive queue.
Fetch is noninteractive, bounded
to one minute, and failures leave scanning enabled with a stale-count warning
in the overview. The CLI collector itself does not fetch.

A `cannot lock ref` fetch failure gets up to three retries, 250 ms apart,
within the same concurrency slot and one-minute timeout. Persistent ref-lock
errors are ignored; independent fetch errors still produce warnings.


## Inventory controls

Settings → Keyboard Shortcuts provides two dashboard hotkeys and a separate
Markdown preview hotkey. See [Markdown preview](../docs/markdown-viewer.md). Control-T
is the default; either recorder can be cleared. Both dashboard shortcuts use the tray toggle: hide a
focused overview, otherwise show and focus it. See [hotkeys](../docs/settings.md#global-hotkeys)
for supported keys, persistence, and native registration limits.

Startup uses mouse mode. Hovering a row chooses it for actions; its identity stays
available when moving to the bottom bar. Up/Down enters keyboard mode and moves
the separately remembered keyboard row. Only keyboard mode shows the cyan
selection and arrow. Esc returns to mouse mode without clearing that row.
Refreshes preserve both identities; missing rows cannot receive stale actions.

Tab switches to the last Markdown preview; Tab in that preview returns here.
Markdown history navigation selects its corresponding dashboard row in keyboard mode.

The bottom hints invoke the same actions as keys: r refreshes the effective
worktree, R refreshes all, a/t/s/f opens agent/ticket/spec/folder. c starts copy:
ct number, cT (also CT) ticket URL, cs spec title, cf folder path. Multiple tickets
use an Up/Down/Enter or clickable chooser. `e` opens the project in the configured editor; `g` opens its Git client. Ctrl-S opens Settings. Q does not quit.

The controller in frontend/controller.mjs is independent of DOM and Wails.
Its Node tests cover selection modes, stable refreshes, action targets, copy
variants, chooser invalidation, and null startup snapshots. Node is test-only.

The blank Uber column shows `u` when spec frontmatter enables `uberreviews: true`.
CLI/MCP AI output uses `-` when disabled.

Q/S combines questions and suggestions; an unknown component keeps the total
unknown. CLI and MCP compact output use the same combined count.

Inventory, Settings and Markdown preview restore position and size independently using a display
UUID and screen-relative frame in macOS user defaults. AppKit frame autosaving
is used only to migrate older settings; see [window placement](../docs/window-placement.md).

The GUI has one Mod column: `0` when clean, `.md` for Markdown-only changes,
and the total changed-file count otherwise. There are no separate New or .md
columns. Known positive Q/S totals use a filled badge; zero is a muted dash.
Ready (`✓`) and input-blocked (`!`) rows have a soft amber background; running
rows use cyan and pushed rows use muted text. See [table model](../docs/table-model.md).

## Missing menu-bar icon on macOS 26

A controlled local test found the icon missing with both direct executable
launches and Launch Services (`open`), using both Wails' default `Item-0` and a
custom `TalkingDuckStatus` autosave name. A brief appearance did not survive
relaunch. Neither launch method nor item identity was a reliable fix.
Control Center logged `Moving host to blocked list`, followed by hiding the
status item, in all four cases. Native `visible` and `window.visible` values
remained true, so they do not prove onscreen visibility. Verify with a screen
capture as well as logs.

[CodexBar issue 1440](https://github.com/steipete/CodexBar/issues/1440) reports the
same symptom from a stale cross-app `menuItemLocations` mapping in Control
Center's `trackedApplications` preferences. A follow-up UI test confirmed
cross-app filtering on this Mac: enabling ChatGPT under Allow in the Menu Bar
immediately showed TalkingDuck; disabling ChatGPT immediately hid TalkingDuck.
TalkingDuck's own switch stayed on throughout. ChatGPT's original off setting
was restored after the test. The exact stored mapping remains unverified:
macOS denied access to that protected preference file.
Do not change bundle IDs, force visibility repeatedly, or reset unrelated
menu-bar settings based on the native visibility flags alone.

The installed bundle now uses `com.andreyvit.talkingduck`, replacing the prototype
`com.andreyvit.talkingduck.mock`. Its first launch through Finder and a subsequent
rebuild/relaunch showed the icon while ChatGPT remained disabled. Changing only
the NSStatusItem autosave name had not fixed the old bundle. A new bundle identity
avoids the old association; it does not repair Control Center's protected record.
Two subsequent restarts through `open` from the agent shell also kept the icon
visible with ChatGPT disabled. `open` is therefore supported for normal restarts.
The original association’s creation was not observed; direct binary launches or
`go run` remain suspected triggers, not proven causes. Avoid those GUI launch
methods rather than risk recreating the association. An old TalkingDuck entry may remain in macOS Menu Bar settings.
Window placement migrates from the prototype preferences; portable settings stay
in their existing file.
