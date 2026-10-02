// Scout prints deterministic Zoo review questions. No agents or product collection.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const header = `Answer each item as:
1. OK <short reason>
2. NA <short reason>
3. FAIL <reason>
Use our question numbers. End with its [check-name] if any. OK = no fix needed, default choice, incl false positives. FAIL = fix needed.`

var codeQuestions = []string{
	"Is there a way to simplify this code while still meeting requirements?",
	"Is a large chunk not obviously correct in isolation? Can it be rewritten more clearly?",
	"Is a business rule spread across pieces? Can it be concentrated?",
	"Does code process many items with avoidable scanning, copying, memory use, or allocations?",
	"Does a new helper, abstraction, type, or interface duplicate one that could be extended?",
	"Are any changes unsafe to deploy without user acknowledgment?",
	"Carefully review all modified code. Are there other significant improvements?",
}

var planQuestions = []string{
	"Is there a simpler end-to-end design that still meets the need?",
	"Is a planned flow or abstraction unclear? Can it be planned more clearly?",
	"Is a business rule spread across planned pieces? Can it be concentrated?",
	"Does the plan scan, allocate, or copy many items when a cheaper approach is as simple?",
	"Does the plan add a helper, type, or mechanism that already exists and could be extended?",
	"Is a planned change unsafe to deploy without acknowledgment in the spec?",
	"Does the spec miss implied edge cases, failures, legacy data, permissions, UI flows, or tests?",
	"Do User request, How it works, Scope, plans, Decisions, or Subtasks contradict each other?",
	"Does a spec claim contradict the code? Code wins.",
	"Does the spec extend scope in a way that would surprise the user?",
}

type change struct{ path, oldPath string }
type check struct {
	name, question string
	group          func(string) bool
}

func productFile(path string) bool {
	// Ranges crossing the directory move still contain the old path.
	if strings.HasPrefix(path, "cmd/scout/") || strings.HasPrefix(path, "scripts/scout/") {
		return false
	}
	switch filepath.Ext(path) {
	case ".go", ".swift", ".m", ".mm", ".js", ".ts", ".tsx", ".py":
		return true
	}
	return false
}

func docFile(path string) bool {
	return path == "AGENTS.md" || path == "README.md" || strings.HasPrefix(path, "docs/") || strings.HasPrefix(path, ".zoo/")
}

var checks = []check{
	{"folder-context", "Do folder, ticket, spec, and owning agent remain associated, including multiple specs and stopped sessions?", productFile},
	{"state-evidence", "Are agent activity, Zoo stage, human attention, priority, and checkout cleanliness distinct, with unknown or stale observations visible?", productFile},
	{"project-docs", "Do the changed docs match current behavior and keep tentative product choices open?", docFile},
}

func git(root string, args ...string) ([]byte, error) {
	args = append([]string{"-c", "diff.noprefix=false", "-c", "diff.mnemonicprefix=false"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func baseRef(root, since string) (string, error) {
	if since == "" {
		return "HEAD", nil
	}
	if since == "upstream" {
		if out, err := git(root, "rev-parse", "--verify", "@{upstream}"); err == nil {
			return strings.TrimSpace(string(out)), nil
		}
		out, err := git(root, "symbolic-ref", "refs/remotes/origin/HEAD")
		if err != nil {
			return "", err
		}
		out, err = git(root, "merge-base", "HEAD", strings.TrimSpace(string(out)))
		return strings.TrimSpace(string(out)), err
	}
	out, err := git(root, "rev-parse", "--verify", "--end-of-options", since+"^{commit}")
	return strings.TrimSpace(string(out)), err
}

func parseChanges(data []byte) ([]change, error) {
	fields := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
	var result []change
	for i := 0; i < len(fields) && fields[i] != ""; {
		if i+1 >= len(fields) {
			return nil, fmt.Errorf("incomplete Git name-status record")
		}
		status, path := fields[i], fields[i+1]
		i += 2
		c := change{path: path}
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			if i >= len(fields) {
				return nil, fmt.Errorf("incomplete Git rename record")
			}
			c.oldPath, c.path = path, fields[i]
			i++
		}
		result = append(result, c)
	}
	return result, nil
}

func loadChanges(root, since string) ([]change, error) {
	base, err := baseRef(root, since)
	if err != nil {
		return nil, err
	}
	out, err := git(root, "diff", "--no-color", "--no-ext-diff", "--no-textconv", "--find-renames", "--name-status", "-z", base, "--")
	if err != nil {
		return nil, err
	}
	changes, err := parseChanges(out)
	if err != nil {
		return nil, err
	}
	out, err = git(root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	for _, path := range strings.Split(string(out), "\x00") {
		if path != "" {
			changes = append(changes, change{path: path})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].path < changes[j].path })
	return changes, nil
}

type options struct {
	plan, highlevel, uber, verbose bool
	since                          string
	selected                       []string
}

func questions(changes []change, opts options) ([]string, error) {
	selected := map[string]bool{}
	for _, name := range opts.selected {
		known := name == "general"
		for _, c := range checks {
			known = known || c.name == name
		}
		if !known {
			return nil, fmt.Errorf("unknown check: %s", name)
		}
		selected[name] = true
	}
	want := func(name string) bool { return len(selected) == 0 || selected[name] }
	var items []string
	if want("general") {
		if opts.plan || opts.highlevel {
			items = append(items, planQuestions...)
			if opts.plan {
				items = append(items, "Does the subtask split cover the plan, order dependencies, and separate unrelated work?")
			}
			items = append(items, "Carefully review the whole spec. Are there other significant improvements?")
		} else {
			items = append(items, codeQuestions...)
		}
	}
	if opts.uber && want("general") {
		if opts.plan || opts.highlevel {
			items = append(items, "Does the whole spec address integration and rollout risks within the current planning stage?")
		} else {
			items = append(items, "Does the entire change meet the request, including integration and deployment risks across subtasks?")
		}
	}
	if opts.plan || opts.highlevel {
		return items, nil // Planning never infers scope from unrelated working changes.
	}
	for _, c := range checks {
		if !want(c.name) {
			continue
		}
		first := true
		for _, f := range changes {
			if !c.group(f.path) && !c.group(f.oldPath) {
				continue
			}
			text := "same q"
			if first {
				text, first = c.question, false
			}
			items = append(items, fmt.Sprintf("%s - %s [%s]", f.path, text, c.name))
		}
	}
	return items, nil
}

func run(args []string, stdout, stderr io.Writer) error {
	opts := options{}
	flags := flag.NewFlagSet("scout", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.BoolVar(&opts.plan, "plan", false, "low-level spec review")
	flags.BoolVar(&opts.highlevel, "highlevel-plan", false, "high-level spec review")
	flags.BoolVar(&opts.uber, "uber", false, "whole-change review")
	flags.BoolVar(&opts.verbose, "v", false, "print scope on stderr")
	flags.StringVar(&opts.since, "since", "", "base commit or upstream; default uncommitted + untracked")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if opts.plan && opts.highlevel {
		return fmt.Errorf("-plan and -highlevel-plan are exclusive")
	}
	opts.selected = flags.Args()
	root, err := git(".", "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	changes, err := loadChanges(strings.TrimSpace(string(root)), opts.since)
	if err != nil {
		return err
	}
	items, err := questions(changes, opts)
	if err != nil {
		return err
	}
	if opts.verbose {
		for _, f := range changes {
			fmt.Fprintf(stderr, "changed: %s\n", f.path)
		}
		for _, c := range checks {
			count := 0
			for _, f := range changes {
				if !opts.plan && !opts.highlevel && (c.group(f.path) || c.group(f.oldPath)) {
					count++
				}
			}
			fmt.Fprintf(stderr, "%s: %d spots\n", c.name, count)
		}
	}
	if len(items) == 0 {
		fmt.Fprintln(stdout, "NO QUESTIONS")
		return nil
	}
	fmt.Fprintln(stdout, header)
	fmt.Fprintln(stdout)
	for i, item := range items {
		fmt.Fprintf(stdout, "%d. %s\n", i+1, item)
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if err == flag.ErrHelp {
			return
		}
		fmt.Fprintln(os.Stderr, "scout:", err)
		os.Exit(1)
	}
}
