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
