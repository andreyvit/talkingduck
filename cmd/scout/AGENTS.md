# Scout work

Follow [DESIGN.md](DESIGN.md) and the installed Zoo Tweak Reviews skill.
While editing scout or review setup: do not run the scout, broad project tests,
or the full Zoo workflow. Run scout tests only. Review the diff with a
`zoo-reviewer` using fallback questions plus the design, then commit the change.

Gate repo questions by matching changed files or exact changed spots. Keep plain
questions only when every review in that mode needs them. Never duplicate a
gated question as a plain question. Checks print questions; they do not run agents.

Fix recurring false positives in code and fixtures. Do not add suppression
comments. Record rejected checks in [REJECTED.md](REJECTED.md).
