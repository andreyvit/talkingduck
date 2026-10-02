---
name: commit
description: Commit completed TalkingDuck changes with a reviewed, scoped diff and relevant validation. Use when asked to commit or when Zoo requires a subtask commit.
---

Follow `.zoo/zoo.md` and file at `$ZOO_LOCAL_MD` if exists.

- Read `git status` and staged/unstaged diffs. Preserve unrelated user work; reconcile pre-existing staged changes before committing.
- Stage explicit paths for completed task or Zoo subtask. Keep ignored evidence, research, credentials, and private agent data out of commits.
- Run relevant checks from `.zoo/testing.md` and project docs. For docs/config only, check syntax, links, paths, and `git diff --check`. Report unavailable checks; never claim they passed.
- Review final staged diff and `git diff --cached --check`. Confirm scope, accidental deletions, secrets, and generated files.
- Write concise imperative subject describing purpose or resulting behavior. Add body only for useful rationale, validation, or limitations. Use ticket ID when task supplies one.
- Commit after required Zoo review passes. Keep one commit per Zoo subtask.
- Verify commit and remaining work with `git show --stat --oneline HEAD` and `git status --short`. Report commit, validation, and any remaining task changes.
- Push only when user requests it.
