# Menu bar app and inventory window

macOS-only Wails v3.0.0-beta.27 prototype. The main window uses live
`inventory.Build` data and the CLI's shared `BuildTable` rows. The menu bar still
uses mock checkout states and the shared `menubarimage` renderer. No frontend
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
`cmd/talkingduck`. The standalone CLI remains cgo-free. The GUI scans on startup and on Refresh,
honoring collection flags and checkout paths. Scans run off the UI thread; overlapping refresh requests are coalesced.
Previous rows remain visible during refresh and after a fatal refresh error.
`--gui=false` overrides the bundle environment when needed.

- Left-click the tray image: hide the overview if visible and focused; otherwise
  show and focus it.
- Right-click: native Wails menu with open, toggle styles, refresh, and quit.
  An explicit `OnRightClick` handler calls `OpenMenu`, bypassing Wails' automatic
  pre-click attachment after a report that it opened only on alternate clicks.
  The suspected cause is attachment timing; repeated manual clicks still need
  confirmation on the affected desktop.
- Standard (default): 123 x 23-point image, 5x7 font, 131-point tray width.
- Compact: 99 x 21-point image, 4x6 font, 105-point tray width.
- Switch layouts live with the native menu checkbox.
  The selection lasts for this run; preferences are not saved yet.
- Closing the window hides it.
- The inventory window displays the same 15 cells and row ordering as the CLI,
  including multiline tickets and partial errors. Monospace text, whitespace,
  and subtle alternating task-row backgrounds separate columns; no gridlines.
  The window contains only the table, status/warnings, and Refresh. Appearance
  controls stay in the tray menu; debug style/image actions are removed.
- The initial mock uses the first listed task per folder from the supplied sample.
  Completed-stage marks and explicit questions produce attention. Unknown activity
  remains unknown. This is not a settled folder-state selection rule.

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
kinds before reading the pointer. Revalidate it before upgrading Wails.

Wails assigns the image asynchronously on the main dispatch queue. Its Go
`InvokeSync` uses a different scheduling mechanism; simply calling it again was
not sufficient at startup. After `SetTemplateIcon`, the shim dispatches its resize
synchronously onto the same native queue. Call `updateWideTray` off the AppKit
main thread, as the Wails event/menu callbacks here do. All mock updates are
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
