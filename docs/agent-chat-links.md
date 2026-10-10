# Opening agent chats

Observed 2026-10-02. Verified a live Claude row opened its matching Code chat.
Codex uses its documented route; automated visual inspection of Codex is blocked
by the UI tool. Agent names link to the selected primary session, not merely
to the application. Links underline on hover and open through macOS URL handling.
Unknown IDs and unsupported agents remain plain text. A conflict marker still
means multiple active primaries; the link follows the same selected primary as
the table. Opening a chat does not send a prompt or start a new task.

## Codex

`codex://threads/<thread-id>` opens an existing local chat. This is documented in
[OpenAI's commands reference](https://learn.chatgpt.com/docs/reference/commands).
The local inventory already supplies its technical thread UUID. CLI-only
sessions are excluded from links. Remote-host and cloud chat routing are not
implemented by this adapter.

## Claude Desktop Code

The installed Claude 2.16120.0 uses
`claude://code/continue?session=<desktop-session-id>` in its own Spotlight and
menu integrations. Its handler looks up that exact session ID and navigates to
its existing route; it does not send a message. This route is observed in the
installed application, not promised by the public deep-link reference. Recheck
it after Claude updates. Archived/unavailable sessions may fall back to the
Code overview.

Use the recorded `app_session_id` (`local_<UUID>`) from desktop metadata or its
session registry. Do not construct this from the CLI transcript UUID: the IDs
can differ. The inventory's primary-session projection now preserves the client
and app-session ID alongside its original engine and transcript/session ID.

Ordinary Claude chats have a separate documented link:
`claude://claude.ai/chat/<conversation-id>`. That conversation ID is not a
Claude Code transcript ID. The current collector tracks Code sessions, so it
does not use this route. See [Claude's deep-link reference](https://support.claude.com/en/articles/14729294-open-claude-desktop-with-a-link).

## Other options

UI search can find a title when a stable deep link is unavailable, but ambiguous
names make that unsuitable for automatic row links. Launching a CLI with resume
flags starts a terminal session rather than selecting the existing desktop chat.
Neither fallback is used here.

## Grok in cmux

Grok links are internal actions, not public URL schemes. On click, read cmux's
`~/.cmuxterm/grok-hook-sessions.json` and find the exact session ID. Reject a
missing record or any newer/equally recent session on the same surface/process.
Verify the PID's start time and Grok executable, then ask cmux's
`agent.resolve_delivery_target` for its live terminal. Require that surface to
match the hook record; select its current workspace, focus the surface, and
activate cmux. No prompt is sent and no session is resumed.

The adapter uses `/Applications/cmux.app/Contents/Resources/bin/cmux`, with a
10-second timeout. Existing socket access must permit TalkingDuck; it does not
change cmux permissions. Missing/ambiguous/stale identities fail visibly. A
session can still change between validation and focus; these APIs are not atomic.

Observed locally: PID resolution returns `source=pid`, `workspace_id`, and
`surface_id`. Saved resume bindings can lag the latest hook session, so they
are not used as identity evidence. Recheck hook schema and RPC support after
cmux upgrades. See [cmux's resolver implementation](https://github.com/manaflow-ai/cmux/blob/main/CLI/CMUXCLI%2BClaudeHookDeliveryTarget.swift).


## Pick an agent

GUI and TUI: `Ax` opens ChatGPT / Codex, `Ac` Claude, `Ag` Grok Build.
Use uppercase `A`, then the agent letter; Escape cancels.
The GUI footer shows `A Pick agent` instead of `a Agent` while Shift is held.
After `A`, both footers show the choices.

Each choice opens the latest primary session associated with that task in that
agent. Reviews are excluded. Missing or unsupported session links show an
unavailable notice; this does not create a new session. Plain `a` still opens
the task's latest primary across all agents.
