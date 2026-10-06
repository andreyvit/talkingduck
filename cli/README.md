# Shared CLI

`cli` owns flags, output modes, terminal width/colors, diagnostics, and exit codes.
It calls `inventory.Build`; it never exits the process or initializes a GUI.

```go
opts := cli.RegisterFlags(flag.CommandLine)
// Both entry points register these same flags, then select their default mode.
flag.Parse()
code := cli.Run(opts, flag.Args())
// The entry point decides when to os.Exit(code).
```

`RunTo(opts, paths, stdout, stderr)` supports captured output and tests. Automatic
colors inspect the supplied output file; buffers/pipes stay plain. CLI parsing
belongs to the caller's `flag.FlagSet`, so repeated invocations do not register
new global flags. Options and snapshots are per invocation.

Exit codes: 0 success, 1 collection/output failure (partial checkout data is still
printed), 2 invalid options. `-ai` prints compact unaligned text with primary session IDs and a final symbol
key, regardless of width/color settings. It takes precedence over `-details`.
There is no `-json` flag; structured snapshots remain in the inventory API.
`cmd/talkingduck/main.go` is the cgo-free entry point. The parallel desktop
prototype can dispatch to the same CLI when GUI mode is not requested.

See [command usage](../cmd/talkingduck/README.md) for all flags and output meanings.

`BuildTable(overview)` returns a UI-independent `Model` with columns, rows, cells,
lines, and actionable inline items, plus legacy string rows and diagnostics.
Only the first row per worktree shows its folder label; every row retains its path
and folder actions. Desktop, TUI, and status output consume the model.
See [the model contract](../docs/table-model.md).

`FormatAI(overview)` renders the same compact text as `-ai` without collection,
for the local MCP server and other callers.

Both entry points register the same flags. `Options.Mode` selects explicit
`-ai`/`-status`/`-tui`/`-gui`, or the caller default. `RunTUI` uses tcell v2
without a widget framework. Input/drawing stay on one goroutine; inventory queues
and open/copy actions run in the background. Tests use tcell simulation screens.

`e` opens the selected worktree in its configured editor; `g` opens its Git client.
GUI and TUI use the same application and ticket settings, including custom commands.
See [launchers](../docs/launchers.md).
