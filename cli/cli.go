// Package cli provides shared flags and terminal output for both entry points.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"talkingduck/inventory"
)

type Options struct {
	Inventory               inventory.Options
	AI, Details, AgentsOnly bool
	Status, TUI, GUI        bool
	MCPPort                 int
	Home, Color             string
	Width                   int
}

// RegisterFlags registers on the caller's FlagSet without parsing it. A desktop
// entry points parse once, select their default mode, then dispatch.
func RegisterFlags(fs *flag.FlagSet) *Options {
	home, _ := os.UserHomeDir()
	o := &Options{}
	fs.BoolVar(&o.Status, "status", false, "print one aligned status table")
	fs.BoolVar(&o.TUI, "tui", false, "run the terminal UI")
	fs.BoolVar(&o.GUI, "gui", false, "run the macOS GUI")
	fs.IntVar(&o.MCPPort, "mcp-port", 42389, "loopback MCP HTTP port (GUI only)")
	fs.StringVar(&o.Inventory.HistoryPath, "state-file", inventory.DefaultHistoryPath(), "persistent task history; empty disables retention")
	fs.StringVar(&o.Inventory.Root, "root", filepath.Join(home, "Developer", "prairie"), "parent of backend and back2 through back10")
	fs.BoolVar(&o.Inventory.NoAgents, "no-agents", false, "skip local agent session collection")
	fs.IntVar(&o.Inventory.AgentLimit, "agent-limit", 0, "recent sessions per provider and folder; 0 = all")
	fs.BoolVar(&o.AI, "ai", false, "compact plain-text inventory for agents with session IDs and a symbol key")
	fs.BoolVar(&o.Details, "details", false, "legacy detailed worktree/session listing")
	fs.IntVar(&o.Width, "width", 0, "table width; 0 uses COLUMNS or 120")
	fs.BoolVar(&o.AgentsOnly, "agents-only", false, "list agent sessions and source warnings as plain text without scanning Git")
	fs.StringVar(&o.Home, "home", home, "home directory for agents-only data discovery")
	fs.StringVar(&o.Color, "color", "auto", "terminal colors: auto, always, never")
	return o
}

// Run prints one inventory and returns an exit code. Only entry points exit.
func Run(opts *Options, paths []string) int { return RunTo(opts, paths, os.Stdout, os.Stderr) }

// RunTo supports callers that capture output. Color auto-detection uses out.
func RunTo(opts *Options, paths []string, out, stderr io.Writer) int {
	if opts.Color != "" && opts.Color != "auto" && opts.Color != "always" && opts.Color != "never" {
		fmt.Fprintln(stderr, "-color must be auto, always, or never")
		return 2
	}
	if opts.Inventory.AgentLimit < 0 {
		fmt.Fprintln(stderr, "-agent-limit must be nonnegative")
		return 2
	}
	fail := func(err error) int { fmt.Fprintln(stderr, err); return 1 }
	if opts.AgentsOnly {
		data, err := inventory.CollectSessions(opts.Home, paths, opts.Inventory.AgentLimit)
		if err != nil {
			return fail(err)
		}
		text, err := renderSessions(data)
		if err != nil {
			return fail(err)
		}
		if _, err = fmt.Fprintln(out, text); err != nil {
			return fail(err)
		}
		return 0
	}
	options := opts.Inventory
	if len(paths) > 0 {
		options.Paths = paths
	}
	snapshot, err := inventory.Build(options)
	if err != nil {
		return fail(err)
	}
	if opts.Details && !opts.AI {
		err = printReport(out, snapshot.Details)
	} else {
		var view tableReport
		view, err = tableData(snapshot.Overview)
		if err == nil {
			var text string
			if opts.AI {
				text = renderAI(view)
			} else {
				tableView, tableErr := BuildTable(snapshot.Overview)
				if tableErr != nil {
					return fail(tableErr)
				}
				text = formatModel(tableView.Model, tableWidth(opts.Width, out), useColor(opts.Color, out))
			}
			_, err = fmt.Fprintln(out, text)
			printWarnings(stderr, view)
		}
	}
	if err != nil {
		return fail(err)
	}
	if snapshot.Failed() {
		return 1
	}
	return 0
}

func printWarnings(w io.Writer, report tableReport) {
	if report.AgentError != "" {
		fmt.Fprintln(w, report.AgentError)
	}
	for _, s := range report.AgentSources {
		if s.Status != "available" {
			fmt.Fprintf(w, "%s: %s coverage; see -agents-only for source warnings\n", s.Engine, s.Status)
		}
	}
	for _, r := range report.Repositories {
		for _, warning := range append(append([]string(nil), r.Warnings...), r.Git.Warnings...) {
			fmt.Fprintf(w, "%s: %s\n", filepath.Base(r.Path), warning)
		}
	}
}
