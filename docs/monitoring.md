# Connection and weekly limits

The menu-bar monitor sits left of project status, at most 53 points wide:
8-point vertical ping text, a 3-point gap, and 42-point quota tracks.
Tracks are 6 points tall with 1-point gaps. Percentages sit in the larger segment;
provider tints appear when multiple tracks are visible. Missing quota is `—`;
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

Ping runs once per second with a 900ms deadline. `20 ms` means a successful reply;
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
