# TalkingDuck

See what needs your attention across agent work folders, tickets, and Zoo stages.

Start with a Go collector for Codex, Claude, and Grok work folders. The final shape
is open: an assistant tool, a Mac view, or later Mac/iPhone apps. Use each stage to
guide the next one.

The first experiment inventories local specs, Git work, and Codex/Claude/Grok sessions:

```sh
go run ./cmd/talkingduck
```

See [inventory usage and limits](cmd/talkingduck/README.md), including JSON output,
and [agent collection research](docs/agent-collection-research.md).
The repository also contains Zoo skills, project guidance, and a review scout.

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
