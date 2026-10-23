# Energy investigation

Measured locally on 2026-10-21 with `sample` and `top`; evidence is under ignored
`.tasks/20261021-energy/`. CPU percentages below describe the main process, not
all WebKit helpers or macOS's Energy Impact score.

## Findings and changes

- A settled 18-second sample showed roughly 6% average CPU, in bursts around
  1–12%. The earlier stack sample was mostly waiting, with inventory collection,
  transcript JSON parsing, filesystem calls, and tray drawing among active stacks.
  This does not establish one dominant CPU consumer.
- `ChangeProbe.Poll` repeatedly resolved the same session working directories
  through symlinks. Cache resolutions within each poll; discard them before the
  next poll so filesystem changes remain visible. A synthetic warm benchmark of
  500 sessions sharing a project improved from 10.15–10.17ms to 2.12–2.13ms per poll.
  This is a component benchmark, not a whole-app energy reduction.
- The Markdown viewer woke every 100ms even while closed. Its timer now exists
  only while the viewer is open. Open-viewer refresh timing stays unchanged.
- The limits strip rebuilt its DOM every second, even for static weekly dates.
  It now sleeps until the countdown boundary, runs no timer while hidden, and
  updates only time-dependent text during visible countdowns/outages. Provider
  snapshots still update immediately.

## Post-change observation

After rebuilding and restarting the main app, the last nine two-second samples
averaged 4.72% main-process CPU (range 1.4–12.2%), versus 6.24% in nine pre-change
samples (1.4–12.3%). Agent workload and UI activity were not controlled, so this
is suggestive only; it does not establish a 24% energy saving. The synthetic
probe benchmark and removed timer work are the reproducible results.

## Remaining costs

Native provider events can still trigger broad session-metadata enumeration;
active projects also request throttled full inventory scans. Full scans and Git
subprocesses remain candidates for a longer CPU profile. The folder-membership
check wakes once per second; process-liveness checks run every 30 seconds.
Claude quota checks launch the CLI every five minutes. These costs were not
changed without evidence that a slower update policy would be worthwhile.

For comparisons, keep monitored folders, visible windows, active agent workload,
and ping settings equal. Ignore startup reconciliation and compare main-process
CPU plus associated WebKit helpers over several minutes. A macOS energy warning
is not an instantaneous CPU measurement; this pass does not claim it is cleared.

## Display sleep and covered windows (2026-10-23)

AppKit workspace notifications now pause background work when all displays sleep
or the Mac sleeps. System wake alone does not resume work: screen wake is required,
so a Power Nap/dark wake does not restart collectors. Startup checks online displays
before starting telemetry. Closing a lid with an active external display keeps
monitoring available there.

- Cancel ping, quota requests/CLI, background Git fetches, settings watches, native
  inventory watches, process checks, discovery and recovery timers.
- Pause both inventory queues, retaining one pending batch per folder. An already
  running inventory collector can finish; its current API is not cancellable.
- On screen wake, restart watchers and reconcile current state through the existing
  queues/throttles. Do not replay every filesystem event missed while asleep.
- AppKit window occlusion controls dashboard DOM work and limits countdowns, including
  windows covered by other apps. Keep only the latest pending snapshot. Markdown
  file polling and rendering also wait while its window is covered/hidden.
- Keep menu-bar monitoring live while displays are awake, even with dashboard hidden.
- Compare tray inputs before pixel drawing and PNG encoding; unchanged inputs skip both.
- Desktop folder membership now uses explicit change notifications, removing the
  one-second membership poll. The CLI-compatible watcher retains its polling API.

Tests cover queue coalescing across pause/resume, collector cancellation, settings
changes while suspended, suspended startup, and explicit membership notifications.
Native lifecycle transitions are logged as `energy: awake=… dashboard=… markdown=…`.
Actual closed-lid/Power Nap behavior still needs a physical sleep/wake observation;
a notification handler and passing unit tests do not establish battery savings.

Tray PNG compression now uses fast compression with reusable buffers. A synthetic
10-cell image benchmark measured 116.7µs and 1.11MB allocated per default encode,
versus 42.3µs and about 30 bytes amortized with pooled fast encoding (64% less time).
Decoded pixels are tested for exact equality. This measures encoding only, not
whole-app energy. Settings also defer folder/archive snapshot construction while
covered or hidden. Native logs confirmed hide/show visibility transitions; reopening
the dashboard and Markdown viewer rendered current content.

Automatic discovery of unconfigured folders runs only while Settings is visible. It has no timer while hidden; reopening Settings scans immediately. Configured projects and menu-bar status continue monitoring while displays are awake.
