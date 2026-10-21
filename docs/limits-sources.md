# Claude and Grok usage sources

Research checked 2026-10-21. Current collectors are documented in
[monitoring.md](monitoring.md); Claude now uses the verified CLI route below. Grok fallback remains unimplemented.

## Reference apps

Installed WhatsMyUsage 1.1.0 contains `CodexBarCore` symbols, including
`GrokRPCClient`, `ClaudeWebSessionKeyRenewalTracker`, and cookie importers.
This establishes shared implementation names, not exact source/version parity.

- [CodexBar Claude sources](https://github.com/steipete/CodexBar/blob/main/docs/claude.md)
- [CodexBar Grok sources](https://github.com/steipete/CodexBar/blob/main/docs/grok.md)
- [TokenBar local Claude reader](https://github.com/wuruoye/TokenBar/blob/main/Sources/TokenBarCore/ClaudeQuotaService.swift)
- [TokenBar local Grok reader](https://github.com/wuruoye/TokenBar/blob/main/Sources/TokenBarCore/GrokQuotaService.swift)

## Claude

WhatsMyUsage preferences recorded `claudeKeychainPromptUsed=true`, no last
Keychain denial, `claudeCLIBackgroundEstablished=true`, and subscription auth.
This supports prior consent and an established CLI route; it does not prove which
route supplied a particular refresh.

A live `claude --print /usage` probe returned the same all-model weekly percentage
and reset time as WhatsMyUsage without a prompt. TalkingDuck now uses this route,
with hooks, MCP, tools, Remote Control startup, and session persistence disabled.
An empty temporary working directory avoids loading project context. Poll every
five minutes; bound execution to twenty seconds and output to 64 KiB. Parse only
the exact all-model weekly line; reject ambiguous percentages, unknown timezones,
and resets outside the coming eight days. No PTY or login automation is needed.
The CLI owns credentials and refresh; TalkingDuck reads neither Claude's stale
credentials file nor its Keychain. Failures clear the reading because CLI usage
output does not establish a stable account identity.

Claude Code file OAuth and Claude Desktop login are separate sources. Signing in
to Desktop does not repair an expired `.claude/.credentials.json` token.
CodexBar also supports OAuth and web session cookies; those alternatives can
require Keychain access. No direct Keychain probe was attempted.

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
