# Review configuration

Scout: `go run ./cmd/scout/scout.go [-highlevel-plan|-plan] [-uber] [-since <ref>|upstream] [-v] [check names...]`

Default: uncommitted and untracked changes. `-since <ref>` includes committed and
working changes since that ref; `-since upstream` uses the tracking branch or
the merge-base with the origin default branch. Errors block the review.

Answer the generated numbered questions. A question is not proof of a violation.
Explain false positives with `OK`; record task-local rejections in the task file.
Fix recurring false positives in scout code and tests; use no suppression comments.
When editing the scout, follow [../cmd/scout/AGENTS.md](../cmd/scout/AGENTS.md)
and use fallback questions for a reviewer instead of running the scout.
