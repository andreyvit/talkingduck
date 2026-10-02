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
Neither fallback is used here. Grok remains unlinked.
