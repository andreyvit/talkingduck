# Window placement

Inventory, Settings and Markdown preview restore independently once their Wails window runtime is
ready, after initial native positioning. Reloading the page does not restore again.
Application-started is too early: Wails starts pending windows asynchronously.

A macOS user-defaults entry stores the display UUID and exact frame relative to
its visible area. AppKit frame autosaving is not registered: its saved tiling
metadata can restore an old untiled width when the window first appears. Move, resize, and key-window notifications update it for visible windows;
fullscreen frames are excluded. Restoring clamps size and position to the target
screen's visible area. A missing display falls back to the current main screen.
Old installations without a display entry migrate the existing AppKit saved frame.

These are local macOS window preferences, separate from portable Zoo settings.

Native regression check (requires macOS):

```sh
clang -fblocks -framework Cocoa -framework CoreGraphics desktop/tests/window-placement.m -o /tmp/td-window-placement-test
/tmp/td-window-placement-test
```

The test uses hidden windows and isolated test keys. With multiple displays it
checks secondary-display restoration; with one it checks that display. It also
checks fallback for a missing display. It does not simulate unplugging a monitor.
