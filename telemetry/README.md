# Connectivity and weekly limits

Independent background collectors; no inventory scans or UI work on their timers.

- Ping: pure-Go unprivileged ICMP, one probe/second, 5s timeout; probes overlap to retain the one-second cadence, and older replies cannot replace newer observations. Match reply payload and source; clear outage timing after a scheduling gap over five seconds. Failure means the selected host did not respond.
- Codex: existing `$CODEX_HOME/auth.json` (default `~/.codex/auth.json`), account usage endpoint, actual seven-day window. Poll every minute.
- Claude: `claude --print /usage`, every five minutes with a 20-second deadline. Claude owns its login and refresh; TalkingDuck never reads Claude credentials or Keychain. Disable hooks, MCP, tools, Remote Control startup, and session persistence; use an empty temporary working directory. Parse only the all-model weekly row and its explicit timezone/reset. No valid row means unavailable. No interactive login or PTY fallback. CLI failures discard the old reading because the output does not identify the account.
- Grok: existing `~/.grok/auth.json`, Build billing endpoint. Accept an explicit percent or a finite used/cap ratio only with a seven-day period. If the primary response omits quota, retry Grok’s bearer gRPC-web billing endpoint within six seconds. Accept an omitted proto3 percentage as zero only with a complete successful response and a current seven-day period. Preserve the primary reset when the periods agree. Other periods, failed RPC status, or malformed data stay unavailable. Poll every minute.

Auto requires session activity in the past 14 days. Stat-only scans inspect Codex `sessions`, Claude `projects`, and Grok `sessions`; JSONL files and Grok `summary.json` qualify. No transcript contents are read. Scans stop at 4,000 entries, six directory levels, or 200ms; a very large tree can yield a false negative. Credential timestamps do not establish activity. On bypasses this gate; Off stops collection.

Requests have a 12-second deadline and reject redirects. Transient failures retain the same credential's last successful value and timestamp as `stale`; auth errors, changed credentials, and logout clear it. Failures back off to 15 minutes; `Retry-After` can extend that delay. Tokens never reach snapshots or logs and TalkingDuck never changes them. Claude manages its own credentials.

`New`, `Update`, `Snapshot`, and `Close` manage collector lifetime. Callbacks run serially outside the state lock; do not call `Close` from a callback. Snapshots own their slice/pointer values. Changing settings cancels old workers and rejects their results.

Validation:

```
go test -race ./telemetry
CGO_ENABLED=0 go test ./telemetry
go vet ./telemetry
```

Tests use synthetic data and local temporary files; they do not contact providers or read real credentials.
