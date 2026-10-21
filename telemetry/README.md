# Connectivity and weekly limits

Independent background collectors; no inventory scans or UI work on their timers.

- Ping: pure-Go unprivileged ICMP, one probe/second, 900ms timeout. Match reply payload and source; clear outage timing after a scheduling gap over five seconds. Failure means the selected host did not respond.
- Codex: existing `$CODEX_HOME/auth.json` (default `~/.codex/auth.json`), account usage endpoint, actual seven-day window. Poll every minute.
- Claude: existing `$CLAUDE_CONFIG_DIR/.credentials.json` (default `~/.claude/.credentials.json`), OAuth `seven_day`. Poll every five minutes. **No Keychain reads, credential prompts, login, or token refresh.** A stale/missing file stays unavailable.
- Grok: existing `~/.grok/auth.json`, Build billing endpoint. Accept an explicit percent or a finite used/cap ratio only with a seven-day period. Missing quota or other periods stay unavailable. Poll every minute.

Auto requires session activity in the past 14 days. Stat-only scans inspect Codex `sessions`, Claude `projects`, and Grok `sessions`; JSONL files and Grok `summary.json` qualify. No transcript contents are read. Scans stop at 4,000 entries, six directory levels, or 200ms; a very large tree can yield a false negative. Credential timestamps do not establish activity. On bypasses this gate; Off stops collection.

Requests have a 12-second deadline and reject redirects. Transient failures retain the same credential's last successful value and timestamp as `stale`; auth errors, changed credentials, and logout clear it. Failures back off to 15 minutes; `Retry-After` can extend that delay. Tokens never reach snapshots or logs and are never changed.

`New`, `Update`, `Snapshot`, and `Close` manage collector lifetime. Callbacks run serially outside the state lock; do not call `Close` from a callback. Snapshots own their slice/pointer values. Changing settings cancels old workers and rejects their results.

Validation:

```
go test -race ./telemetry
CGO_ENABLED=0 go test ./telemetry
go vet ./telemetry
```

Tests use synthetic data and local temporary files; they do not contact providers or read real credentials.
