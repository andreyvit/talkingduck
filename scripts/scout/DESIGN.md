# Scout design

`scout.go` prints deterministic, numbered Zoo review questions. Standalone,
standard-library Go matches the initial project direction without a module.
Keep it small and easy to replace as the product moves toward Swift.
It never runs agents, calls an LLM, or prints diffs. Tests call its functions;
run them using the command in `docs/development.md`.

## CLI and output

`go run ./scripts/scout/scout.go [-highlevel-plan|-plan] [-uber] [-since <ref>|upstream] [-v] [check names...]`

- Default: implementation review of uncommitted plus untracked files.
- `-highlevel-plan`: high-level spec review; `-plan`: low-level spec review.
  These modes are exclusive. They never gate questions on working code changes.
- `-uber`: review the whole change across subtasks; combines with planning modes.
- `-since <ref>`: committed and working changes since a verified commit.
- `-since upstream`: tracking branch, otherwise merge-base with origin's recorded
  default branch. Missing refs fail; the tool never fetches or guesses a branch.
- `-v`: changed paths and gate/spot counts on stderr.
- Check names select `general`, `folder-context`, `state-evidence`, or `project-docs`.
  Unknown names fail. No selected questions: exactly `NO QUESTIONS`.

Print answer instructions, blank line, then numbered items. Plain items precede
gated items. Spots sort by path; repeated question text becomes `same q` with the
same check name. Errors exit nonzero before printing any successful question list.

## Pipeline

Read Git diff and untracked paths → register mode questions → gate by changed
file groups → attach changed-file spots → print. The current checks are judgment
questions about whole files. They use path-only pointers, including deleted,
binary, empty, huge files and symlinks. No file bodies are read.

Git format is pinned: prefixes, color, external diff and text conversion.
Parse NUL-delimited names for spaces, Unicode, and renames. Keep old and new paths
for gates; deleted files still count. Untracked files count as changed paths.
Never follow untracked symlinks. No hunk parser or syntax analysis is needed yet.
`.spec/` does not match current gates; active task files produce no code questions.

File groups are small predicates by extension and path prefix. Scout sources
and fixtures are excluded from product-code groups. Test group boundaries,
modes, renames, deletions, errors,
scope, and likely false positives with synthetic files and temporary Git repos.

## Questions and extensions

Plain mode questions come from Zoo's spec/code fallback lists. Repo-specific
questions come from `docs/product.md` and `docs/development.md`. No typed-code or
text-violation checks exist yet; do not imply syntax/type analysis.

Add precise spot sources before broad ones. Checkable rule: violation at a spot.
Judgment: question at a spot or whole-file gate. Neither: plain mode question if
universal, otherwise document it. Keep each check's name, group, and static
question together. Checks use supplied changes/readers; never call Git inside a
check. Add behavior fixtures for hits, misses, and likely false positives.

Future diagnostics survive only on added lines; region questions survive when
their region overlaps added lines or deletion anchors. Diff questions may inspect
removed lines directly. Skip formatter-only remove/re-add realignment. Load AST
or type analysis only when a relevant source file changes. Introduce such machinery
only when a real check requires it.

Reviewer `OK` answers with reasons are false-positive evidence. Record task-local
rejections in task files; fix recurring ones with narrower matches, exclusions,
recognized idioms, changed-field filtering, or pushed-state checks. No suppression
markers. Compare compatibility against pushed code when a future check needs it.
