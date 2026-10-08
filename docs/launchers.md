# Project and ticket launchers

`launchers/` is a standalone Go package: application catalogs, platform filters,
command planning, argument parsing, launch execution, and ticket URL templates.
It imports no inventory, UI, or settings code. `settings/` persists choices;
`actions/` joins settings with table actions.

## Applications

`e` opens the row's worktree in the editor; `g` opens it in the Git client.
Defaults: Zed and Sublime Merge. Settings offers supported applications for the
current OS plus Custom. An offered application need not be installed. Missing
launchers produce an error; selecting a preset does not install software.

macOS presets use bundle IDs or a bundled command-line launcher. CLI presets
use PATH; macOS/Linux also check common Homebrew/local executable directories.
Windows command presets require the executable on PATH; registry discovery is
not implemented. Linux/Windows commands are source-checked, not live-tested.
The launcher package cross-compiles; the entire monitor is not claimed portable.

Custom takes an executable path (or a macOS `.app`) and arguments. `$PATH` is
the selected worktree, not the environment variable. Quotes group arguments;
backslashes escape quotes, backslashes, or whitespace. Other backslashes remain
literal for Windows paths. Parsing precedes substitution, preserving spaces in
paths as one argument. No shell, glob, command substitution, or environment
expansion. `~/` is supported in executable paths. A custom `.app` uses a new
instance so macOS delivers its arguments even if already running.

Example executable `/usr/local/bin/editor`, arguments `--project "$PATH"`.
For a custom macOS application receiving files through Open Documents, choose
`/usr/bin/open` with arguments `-a "/Applications/Example.app" "$PATH"`.

## Ticket trackers

Presets: Linear, Jira Cloud/Server, GitHub Issues, GitLab Issues, Azure DevOps,
YouTrack, Redmine, Shortcut, Trello, Custom. Their definitions contain `name`,
`ticketURL`, `rootURL`, and `urlRegExp` (plus a stable ID).

A regexp exposes a workspace/project URL field. Pasting a URL validates the
whole URL and saves capture group 1 as `account_slug`. `$ACCOUNT_SLUG` fills
both URL templates; `$TICKET` fills the ticket link with a percent-encoded ID.
The account may represent a workspace, repository path, or self-hosted base URL.
Open-ticket and copy-ticket-URL use the same resolver and current settings.
Copy-ticket-ID remains unchanged. No tracker authentication is needed.

Custom accepts a ticket URL template, optional root URL, and optional Go regexp.
Use `$TICKET` in the ticket template; a regexp with group 1 is required when
using `$ACCOUNT_SLUG`. Self-hosted subpaths other than the preset's conventional
`/jira`, `/youtrack`, or `/redmine` should use Custom. GitLab accepts nested
project groups; paste the project root or a URL containing `/-/`.
Trello uses the card's short ID, not the board-local card number.

These preferences configure linking, not spec ticket discovery. Existing
inventory extraction still primarily recognizes Zoo-style `ABC-123` IDs; choosing
a tracker does not infer numeric ticket IDs from arbitrary prose.

## Research and maintenance

Checked 2026-10-06. Catalog entries are factual integration data, not copied
implementation code. No binaries needed reverse engineering.

- [GitHub Desktop macOS editor catalog](https://github.com/desktop/desktop/blob/development/app/src/lib/editors/darwin.ts)
  supplies common editor names and bundle IDs. We omit obsolete/specialist entries.
  Its [Linux catalog](https://github.com/desktop/desktop/blob/development/app/src/lib/editors/linux.ts)
  and [Windows catalog](https://github.com/desktop/desktop/blob/development/app/src/lib/editors/win32.ts)
  identify executable names. Its [launcher](https://github.com/desktop/desktop/blob/development/app/src/lib/editors/launch.ts)
  also separates executable, arguments, and project-path substitution.
- [Raycast Git Repos](https://github.com/raycast/extensions/tree/main/extensions/git-repos)
  uses application pickers for opening repositories; its list is installed apps,
  not a portable hardcoded Git-client catalog. [Tower Repositories](https://www.raycast.com/tdwesten/tower)
  is another repository launcher example.
- [Sublime Merge CLI](https://www.sublimemerge.com/docs/command_line),
  [Tower CLI](https://www.git-tower.com/help/guides/integration/using-the-cli-tool/mac),
  and [GitKraken CLI invocation](https://feedback.gitkraken.com/suggestions/366462/add-a-feature-to-open-a-project-tab-in-gitkraken-from-the-terminal)
  document project opening. [Homebrew casks](https://github.com/Homebrew/homebrew-cask/tree/master/Casks)
  confirm Fork/Tower/Sourcetree/GitKraken bundle IDs and installed launcher paths.
  Local Zed and Sublime Merge Info.plist files confirmed their bundle IDs.
- [GitKraken tracker integrations](https://help.gitkraken.com/kepler/issue-tracker-integrations/)
  and [JetBrains task management](https://www.jetbrains.com/help/idea/managing-tasks-and-context.html)
  informed the tracker list. Templates use the services' issue URL routes;
  tests cover pasted roots, deep links, nested groups, and rejected hosts.

When changing a preset, check vendor source/docs and add a command-plan or URL
example. Never silently fall back to an unrelated application or tracker.

## Markdown viewers

`applications.markdown` uses the same selection/custom-command schema. `(Default)`
opens a spec with the OS file association; `(Text editor)` uses the configured
editor and passes the spec file instead of its folder. Custom `$PATH` is the file.
The viewer catalog is platform-filtered: MarkDownReader, Marked, MacMD Viewer,
MacDown (Mac); Typora and MarkText (Mac/Linux/Windows); ghostwriter (Linux/Windows).
App names sort alphabetically after the two special choices; Custom stays last.
Editor and Git client catalogs use the same alphabetical ordering.

Settings checks standard Mac application directories (including Setapp and
`~/Applications`) or executable paths elsewhere once per launch. Missing apps
use native disabled styling; Custom remains available. Detection is a hint:
portable apps, renamed bundles and nonstandard installation paths can be missed.
No Spotlight query, recursive scan, app launch or network request is involved.
On Mac, a detected app bundle path takes precedence over the catalog bundle ID,
so Setapp variants work without assuming their bundle ID matches the direct app.

Verified IDs and sources (2026-10-08):

- MarkDownReader: `com.markdownreader.MarkDownReader`; local installed Info.plist.
  This is distinct from similarly named MarkdownReader projects online.
- Marked: `com.brettterpstra.marked.pro`; local installed Info.plist.
- MacMD Viewer: `com.arthur.MarkdownViewer`, [Homebrew cask](https://github.com/Homebrew/homebrew-cask/blob/master/Casks/m/macmd-viewer.rb).
  [Vendor integration docs](https://macmdviewer.com/docs/integrations) and
  [Setapp listing](https://setapp.com/apps/macmd-viewer) confirm document-reader use.
- MacDown: `com.uranusjr.macdown`, [Homebrew cask](https://github.com/Homebrew/homebrew-cask/blob/master/Casks/m/macdown.rb).
- Typora: `abnerworks.Typora`, [Homebrew cask](https://github.com/Homebrew/homebrew-cask/blob/master/Casks/t/typora.rb);
  [vendor CLI instructions](https://support.typora.io/Use-Typora-From-Shell-or-cmd/).
- MarkText: `com.github.marktext.marktext`, [Homebrew cask](https://github.com/Homebrew/homebrew-cask/blob/master/Casks/m/mark-text.rb)
  and [upstream platform support](https://github.com/marktext/marktext).
- ghostwriter: [upstream README and command-line usage](https://github.com/KDE/ghostwriter/blob/master/README.md).

The catalogs select document-oriented tools, following existing Homebrew and
vendor listings. Linux/Windows commands are source-checked, not locally tested;
missing command-line launchers can be supplied through Custom.
