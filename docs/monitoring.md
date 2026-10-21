# Connection and weekly limits

The menu-bar monitor sits left of project status, at most 24 points wide:
6-point vertical ping text, a 1-point gap, and 17-point quota tracks.
A 1-point gap separates monitoring from projects.
Tracks have 1-point gaps and softly cut corners; their outer edges align with
the project grid. Remaining values use digits only and sit in the larger segment. Near-white provider tints appear when multiple tracks are visible. Missing quota is `—`;
a stale quota carries `~`. Project glyphs retain native appearance colors.

Settings → Menu Bar:

- Ping: Vertical text (default) / Off.
- Server: Google 8.8.8.8 (default), Cloudflare 1.1.1.1, Quad9 9.9.9.9, Custom hostname/IP.
- Each provider: On / Auto (default) / Off. Auto means activity in the past 14 days,
  determined from session-file timestamps, without reading credentials for visibility.
- Projects: Projects with menu bar shortcut (default) / Off. Hiding project status
  does not stop inventory monitoring.

`l Limits` toggles a fixed strip above the dashboard command bar. Its visibility
persists locally. Each provider shows remaining percentage and a reset label aligned
with the progress bar's left edge. Resets within 24 hours use a countdown; later
resets use local weekday/time. Unknown and expired readings never imply a fresh quota.

Monitor text uses integer-grid pixel fonts, including tightly kerned `ms` and a
1-point word space. Ping is centered in a field sized for the largest reading
in the last ten seconds: 1 digit below 10ms, 2 from 10ms, 3 from 100ms, 4 from
1000ms. Fields expand immediately and shrink when the larger sample expires. The main
window uses the same width with a three-digit minimum and right-aligned numbers.
Ping is white below 50ms, bright yellow from 50ms, and bright red from 150ms
(also red during outages). Native smoothing is disabled.

Ping runs once per second with a 5-second deadline, allowing four-digit latency readings. `20 ms` means a successful reply;
`3 s` means the selected server has not replied for three seconds. A failed target
is not proof the entire internet is down. Scheduling gaps over five seconds reset
outage timing, including sleep/wake. Configuration errors are validated before saving.

Collectors run independently of inventory scans. Codex/Grok poll every minute;
Claude polls every five minutes. Requests time out after 12 seconds, back off on
failure, and honor Retry-After. Off stops that collector. Existing local auth files
can authorize read-only usage requests; the app never reads Keychain items, asks for
login, or refreshes/writes tokens. Unsupported or inaccessible quotas remain unavailable.
See [collector details](../telemetry/README.md).

Initial local verification: ping and Codex weekly usage worked; Claude's local token
returned HTTP 401; Grok billing responded but supplied no usable seven-day quota.
These are observations from one account setup, not provider-wide guarantees.
