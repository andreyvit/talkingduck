# Shared CLI

`cli` owns flags, output modes, terminal width/colors, diagnostics, and exit codes.
It calls `inventory.Build`; it never exits the process or initializes a GUI.

```go
opts := cli.RegisterFlags(flag.CommandLine)
// A desktop entry point can register --gui here, then parse once.
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

`BuildTable(overview)` returns the same unwrapped 15-column task rows used by
the terminal renderer, plus diagnostics and row-aligned absolute spec and worktree paths for links.
Only the first row per worktree shows its folder label; every row retains its absolute path. The desktop window consumes these cells
directly; multiline tickets remain within one task row. It adds no collection
logic or state inference.

`FormatAI(overview)` renders the same compact text as `-ai` without collection,
for the local MCP server and other callers.
