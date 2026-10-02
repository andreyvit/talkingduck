# TalkingDuck

See what needs your attention across agent work folders, tickets, and Zoo stages.

The goal is Mac and iPhone apps for managing Codex, Claude, and Grok work. Start
small, use each stage, and let that experience guide the next one. Start with Go;
the repository will likely move toward Swift over time.

This repository currently contains Zoo skills, project guidance, and a review
scout. Product code has not started.

- [Product vision](docs/product.md)
- [AI documentation](docs/README.md)
- [Development and validation](docs/development.md)
- [Agent instructions](AGENTS.md)

Zoo skills live in `.agents/skills`; `.claude/skills` links to that directory.
Codex and Claude reviewer, researcher, and browser roles are installed too.
To refresh the standard skills from the sibling Zoo checkout:

```sh
../zoo/install.sh "$PWD"
```

Project customization lives in `.zoo/`. The `commit` skill is repo-specific.
