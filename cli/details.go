package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"talkingduck/inventory"
	"time"
)

func renderDetails(out io.Writer, report inventory.Report) {
	fmt.Fprintf(out, "Local work inventory — %s\n%s\n", report.ObservedAt.Format(time.RFC3339), report.Boundary)
	if report.AgentError != "" {
		fmt.Fprintln(out, "Agent collection:", report.AgentError)
	}
	for _, raw := range report.AgentSources {
		var source struct {
			Engine   string   `json:"engine"`
			Status   string   `json:"status"`
			Warnings []string `json:"warnings"`
		}
		if json.Unmarshal(raw, &source) == nil {
			if source.Status != "available" {
				fmt.Fprintf(out, "Agent source %s: %s\n", source.Engine, source.Status)
			}
			for _, w := range source.Warnings {
				fmt.Fprintf(out, "  %s: %s\n", source.Engine, w)
			}
		}
	}
	for _, r := range report.Repositories {
		fmt.Fprintf(out, "\n%s [%s] — %d unpushed commits\n  %s\n", filepath.Base(r.Path), r.Branch, len(r.Commits), r.Path)
		if r.Error != "" {
			fmt.Fprintln(out, "  ERROR:", r.Error)
			continue
		}
		for _, raw := range r.Agents {
			var a inventory.AgentSession
			if json.Unmarshal(raw, &a) == nil {
				fmt.Fprintf(out, "  Agent %s | %s | activity: %s | question: %s\n", a.Engine, a.Title, a.Activity.Value, a.Question.Value)
				if a.LastMessage.At != nil {
					fmt.Fprintln(out, "    Last text message:", *a.LastMessage.At)
				}
				for _, q := range a.PendingQuestions {
					for _, prompt := range q.Prompts {
						fmt.Fprintln(out, "    Question:", prompt)
					}
				}
			}
		}
		if len(r.Specs) == 0 {
			fmt.Fprintln(out, "  No specs with local work found.")
		}
		for _, s := range r.Specs {
			ts := strings.Join(s.Tickets, ", ")
			if ts == "" {
				ts = "no ticket found"
			}
			activity := "unknown"
			if !s.LastActivity.IsZero() {
				activity = s.LastActivity.Local().Format("2006-01-02 15:04")
			}
			fmt.Fprintf(out, "  %s | %s | %s\n    %s\n    status: %s\n    %d commits; spec dirty: %t; %d associated dirty files; last: %s\n", ts, s.Stage, s.Title, s.Path, s.Status, len(s.Commits), s.Dirty, len(s.DirtyFiles), activity)
			if s.GitStatus != "" {
				fmt.Fprintf(out, "    Git status: %q (?? = untracked)\n", s.GitStatus)
			}
			if s.Deleted {
				fmt.Fprintln(out, "    spec deleted; metadata from HEAD when available")
			}
			inferred := false
			for _, e := range s.Evidence {
				inferred = inferred || (e.Inferred && e.Kind != "tickets inferred from filename and request")
			}
			if inferred {
				fmt.Fprintln(out, "    includes inferred file/ticket associations; see -json for evidence")
			}
		}
		if len(r.UnassignedCommits) > 0 {
			fmt.Fprintf(out, "  Unassigned commits: %d\n", len(r.UnassignedCommits))
			for _, c := range r.Commits {
				if slices.Contains(r.UnassignedCommits, c.Hash) {
					fmt.Fprintf(out, "    %s %s\n", c.Hash[:10], c.Subject)
				}
			}
		}
		for _, p := range r.UnassignedFiles {
			fmt.Fprintln(out, "  Unassigned file:", p)
		}
		for _, p := range r.AuxiliaryFiles {
			fmt.Fprintln(out, "  Auxiliary file (no activity boost):", p)
		}
		for _, w := range r.Warnings {
			fmt.Fprintln(out, "  WARNING:", w)
		}
	}
}

func printReport(out io.Writer, report inventory.Report) error {
	var b strings.Builder
	renderDetails(&b, report)
	_, err := io.WriteString(out, b.String())
	return err
}
