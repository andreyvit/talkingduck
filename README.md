# TalkingDuck

See what needs your attention across agent work folders, tickets, and Zoo stages.

Start with a Go collector for Codex, Claude, and Grok work folders. The final shape
is open: an assistant tool, a Mac view, or later Mac/iPhone apps. Use each stage to
guide the next one.

## Mac app

Requires Go and Xcode Command Line Tools. From the repository root:

```sh
./desktop/build.sh
open "$HOME/Applications/TalkingDuck.app"
```

The build installs an ad-hoc-signed app into `~/Applications`. Launch it from
Finder, Spotlight, or the command above. Click its menu-bar icon or press Ctrl-T
to toggle the inventory while it runs. Quit before rebuilding, then launch again.
No login-item setup or notarization is included.

See [desktop details and validation](desktop/README.md).

## CLI

The CLI opens the terminal UI by default. Both binaries accept the same mode flags:

```sh
go run ./cmd/talkingduck          # TUI (default)
go run ./cmd/talkingduck -status  # one-shot aligned table
go run ./cmd/talkingduck -ai      # compact text for agents
go run ./cmd/talkingduck -gui     # installed Mac app
```

In the TUI: ↑/↓ selects, `r` refreshes the selected worktree, `R` refreshes all.
Use `a` agent, `t` ticket, `s` spec, `f` folder; `ct` ticket number, `cs` spec
title, `cf` folder path. Multiple tickets open a chooser. `g` is reserved.
Esc cancels a chord/chooser; Shift-Q or Ctrl-C quits. Titles wrap; there is no horizontal scrolling.
Mouse: click rows to select, links to open, or hover and click `©` to copy.
The wheel scrolls; keyboard selection brings its row back into view.

See [inventory usage and limits](cmd/talkingduck/README.md), including compact `-ai` output,
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

Keyboard: `cT` copies the ticket URL; `ct` copies its number. Multiple tickets
use the same chooser. GUI also accepts `CT`.

Shift-X archives the effective row permanently in the state file (GUI and TUI).
Task rows are identified by worktree/spec; a task-free row is archived separately.
No Git files are changed. Running tasks stay archived too.
