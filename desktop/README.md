# Menu bar app and inventory window

macOS-only Wails v3.0.0-beta.27 prototype. The main window uses live
`inventory.Inventory` cache and the CLI's shared `BuildTable` rows. The menu bar
uses the same cached checkout states and the shared `menubarimage` renderer. No frontend
dependencies, Node, generated bindings, or external web server.

From the repository root:

```sh
./desktop/build.sh
open .tasks/desktop/TalkingDuck.app
# Or launch the built executable directly:
.tasks/desktop/TalkingDuck.app/Contents/MacOS/talkingduck-desktop --gui
```

Requires Xcode Command Line Tools. The build script compiles with cgo, targets
macOS 13+, packages the app, and signs it ad hoc for local use. It does not
notarize, install, or register autostart. Generated output stays under `.tasks/`.
The bundle's `LSEnvironment` sets `TALKINGDUCK_GUI=1`; direct execution requires
`--gui`. Without GUI mode the desktop executable calls the shared `cli.Run`, just like
`cmd/talkingduck`. The standalone CLI remains cgo-free. The GUI creates one in-memory cache and
scans each configured checkout sequentially, publishing after each folder.
After a full pass finishes it waits one minute, then repeats while running,
even with the window hidden. Refresh starts a pass immediately during the pause;
requests during an active pass coalesce. Collection flags and paths apply.
Previous rows remain visible during refresh; failed folders retain their last
successful data and show a warning. Unscanned folders show explicit placeholders.
The status names the current folder and the latest completed pass, not a shared
observation time for every row. Quit stops scheduling; an in-flight collector
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
- Click any ticket reference to open its Linear issue in the default URL handler.
  Each ticket on a multiline row is linked separately; links underline on hover.
  The workspace is currently Bubblehouse; see [future settings](../docs/future-settings.md).
- Click a spec title to preview its file in Marked. Titles underline on hover;
  non-task rows stay plain text. Missing files or app launch errors appear in the window.
- Closing the window hides it.
- The inventory window displays the same 15 cells and row ordering as the CLI,
  including multiline tickets and partial errors. Monospace text, whitespace,
  and subtle alternating task-row backgrounds separate columns; no gridlines.
  The window contains only the table, status/warnings, and Refresh. Appearance
  controls stay in the tray menu; debug style/image actions are removed.
- Tray numbers follow configured checkout order. The first listed task supplies
  the stage, preserving the prototype's selection rule. Explicit questions or
  completed turns need attention; running turns get an outline. Failed/pending
  entries are unknown. Task-free checkouts show `-` when clean and `d` when
  modified (including informational Markdown changes). Multiple-task priority
  remains a future decision.

## Local MCP

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

Wails assigns the image asynchronously on the main dispatch queue. Its Go
`InvokeSync` uses a different scheduling mechanism; simply calling it again was
not sufficient at startup. After `SetTemplateIcon`, the shim dispatches its resize
synchronously onto the same native queue. Call `updateWideTray` off the AppKit
main thread, as the Wails event/menu callbacks here do. All image updates are
serialized. Native checks verify image width, height, and item length.

Always update images through this adapter: calling Wails' setter alone restores
the square. An unknown Wails version fails visibly instead of silently reading
an assumed memory layout. A public native-handle or icon-size API would remove
this dependency.

## Validation

```sh
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
