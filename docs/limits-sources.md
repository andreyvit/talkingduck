# Claude and Grok usage sources

Research checked 2026-10-21. Current collectors are documented in
[monitoring.md](monitoring.md); the alternatives below are not yet implemented.

## Reference apps

Installed WhatsMyUsage 1.1.0 contains `CodexBarCore` symbols, including
`GrokRPCClient`, `ClaudeWebSessionKeyRenewalTracker`, and cookie importers.
This establishes shared implementation names, not exact source/version parity.

- [CodexBar Claude sources](https://github.com/steipete/CodexBar/blob/main/docs/claude.md)
- [CodexBar Grok sources](https://github.com/steipete/CodexBar/blob/main/docs/grok.md)
- [TokenBar local Claude reader](https://github.com/wuruoye/TokenBar/blob/main/Sources/TokenBarCore/ClaudeQuotaService.swift)
- [TokenBar local Grok reader](https://github.com/wuruoye/TokenBar/blob/main/Sources/TokenBarCore/GrokQuotaService.swift)

## Claude

Claude Code file OAuth and Claude Desktop login are separate sources. Signing in
to Desktop does not repair an expired `.claude/.credentials.json` token.
CodexBar supports OAuth, CLI probing, and web session cookies. Desktop cookie
import reads `Library/Application Support/Claude/Cookies` and decrypts through
`Claude Safe Storage` / `Claude Key`; this can require Keychain permission.
Background polling must never prompt. An explicit connection action remains a
product decision; no Keychain access was attempted during this research.

TokenBar reads `Library/Application Support/Claude/plan-usage-history.json`:
`samples[].t` is epoch milliseconds; `samples[].u.sd` is weekly used percent,
`fh` is five-hour used percent. It accepts samples younger than 30 minutes.
The file does not supply reset timestamps. On the inspected installation the
newest sample was twelve days old, while Desktop's live Usage popover showed a
new weekly reading. Opening that popover did not refresh the history file.
Therefore this file alone is not a reliable live Desktop integration.

## Grok

The installed CLI accepts newline-delimited JSON-RPC over `grok agent stdio`:

1. `initialize` with `protocolVersion: "1"`, filesystem capabilities false,
   and `terminal: false`.
2. `_x.ai/billing` with **`params: {}`**. Omitting params returned `-32602`.

This delegates authentication to Grok without reading credentials ourselves.
The successful response can still omit `creditUsagePercent`, with both
`onDemandCap.val` and `onDemandUsed.val` zero. This happened in the live probe.
Zero divided by zero is unknown, not an unused quota.

`~/.grok/logs/unified.jsonl` also records `billing: fetched credits config` in
`ctx.config`. Recent records showed the same omission. Older percentages from
a previous billing period must not fill a current-period gap.

CodexBar retries a bearer-authenticated gRPC-web billing endpoint:
`https://grok.com/grok_api_v2.GrokBuildBilling/GetGrokCreditsConfig`.
The request frame is `00 00 00 00 02 08 00`; field 1 is an explicit false
`exclude_legacy_monthly_usage`, not a period selector. Reject redirects.
A live read-only probe succeeded with HTTP 200 and gRPC status 0, without cookies
or Keychain access. The response had a current weekly period and no float field.

CodexBar treats an omitted proto3 percentage as zero only when a complete,
valid response has a recognized current period containing now. It tracks this
as an implicit zero, separate from a published percentage. Preserve the proxy's
authoritative reset. A missing JSON percentage alone is insufficient. Parse
only schema-known nested messages; unknown bytes are not nested protobuf.
Check gRPC trailers, malformed lengths, overflowing varints, expired periods,
and explicit nonzero percentages in synthetic tests before implementation.
