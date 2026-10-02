# Product vision

TalkingDuck should answer: **What needs my attention next, where is that work,
and which agent is handling it?**

The destination is Mac and iPhone apps for managing agent work, with Zoo as the
main workflow. Initial agents: Codex, Claude, and Grok. Build slowly in useful
stages; daily use should guide what comes next.

## Problem

Work spans many folders. Andrey uses `backend`, `back2` through `back10`, and may
add more. Other users may have many projects or worktrees. Folder names and counts
must not be fixed in the product.

Opening folders one by one makes it hard to find a ticket, spot a finished change
awaiting review, answer a question, approve a spec, or know which folder is free.
A chat list and an agent process list each show only part of this context.

## What needs to stay visible

- Full folder identity, with a short name for display. Separate checkouts of one
  repository remain distinct.
- Ticket number and title or a short summary, linked to its spec and folder.
- Agent/engine and session handling the work.
- Zoo stage: high-level planning, low-level planning, spec review, uber-review,
  execution, final review, and completion, including approval boundaries.
- What requires human action: question, spec approval, pending suggestion, result
  review, or another explicit next step.
- Agent activity, separately: running, waiting, stopped, or unknown.
- Folder/work status: clean or changed checkout; specs written but waiting;
  work completed but still needing review; no active execution.

A folder can hold several specs in parallel. Only one task can execute in that
folder at once; other specs, parked work, and review activity remain visible.
Show an ownership conflict if observations disagree rather than selecting a
winner silently.

## Views

Keep a persistent folder overview alongside ways to find work by ticket and
urgency. Urgent tickets must be findable across folders. Each result still shows
its folder, spec, agent, stage, and next action.

Urgency and need for attention are separate: a question on a low-priority ticket
does not establish that ticket's business priority. Priority source and sorting
rules remain open decisions.

Finished or stopped sessions must not erase their folder or reviewable work.
A clean Git checkout does not establish that a task is finished. A completed
agent turn does not establish that Zoo execution is complete.

## Client coverage and reliable state

Desktop Codex/ChatGPT and desktop Claude are required. Andrey uses cmux for
Grok Build; terminal-only monitoring would miss much of his work.

Prefer explicit session-to-spec associations over selecting the most recently
edited spec. File recency and free-form spec status can provide clues, but must
not silently establish ownership or an exact workflow stage. Show unknown,
ambiguous, and stale observations honestly, with their source and update time.

## Incremental approach

Start with Go, likely a small script that collects and reports information.
Its scope, sources, and output have not been decided. Over time, the repository
will likely become Swift, rewriting early pieces as useful. Keep the scout small
and nimble too. Use each result to learn before building more UI or automation.

Open decisions: first collector scope; agent integration mechanisms; explicit Zoo
status publishing; ticket priority sources; storage; Mac-to-iPhone transport;
notifications; and whether to reuse an existing monitor. This initialization
does not select a framework, service, database, or external monitor.

## Background

Captured from the user's repository brief and follow-up on 2026-10-02. The chat
**Find agent monitoring apps** (2026-09-30) also established desktop coverage,
persistent folder rows, and separate agent/Zoo states. That chat explored cctop
as a possible base; it did not test it live or implement an integration.
Current chat listings confirm work spread across multiple Prairie folders and
several chats within a folder. They do not establish which task is executing now.
