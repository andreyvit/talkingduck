package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func command(t *testing.T, dir string, args ...string) string {
	t.Helper()
	s, e := git(dir, args...)
	if e != nil {
		t.Fatal(e)
	}
	return strings.TrimSpace(s)
}
func write(t *testing.T, dir, path, body string) {
	t.Helper()
	p := filepath.Join(dir, path)
	if e := os.MkdirAll(filepath.Dir(p), 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(body), 0644); e != nil {
		t.Fatal(e)
	}
}
func commit(t *testing.T, dir, msg string) string {
	t.Helper()
	command(t, dir, "add", ".")
	command(t, dir, "commit", "-qm", msg)
	return command(t, dir, "rev-parse", "HEAD")
}
func repo(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	command(t, d, "init", "-q")
	command(t, d, "config", "user.name", "Test")
	command(t, d, "config", "user.email", "test@example.invalid")
	command(t, d, "config", "commit.gpgsign", "false")
	write(t, d, "base.txt", "base")
	h := commit(t, d, "base")
	command(t, d, "update-ref", "refs/remotes/origin/main", h)
	return d
}
func checked(t *testing.T, d string) Repo {
	t.Helper()
	r := scan(d, time.Now())
	if r.Error != "" {
		t.Fatal(r.Error)
	}
	return r
}
func TestSpecInstructionsAreNotTasks(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/AGENTS.md", "# Spec structure\n")
	write(t, d, ".spec/current.md", "---\nstatus: planning\n---\n# Current task\n")
	r := checked(t, d)
	if len(r.Specs) != 1 || r.Specs[0].Path != ".spec/current.md" {
		t.Fatalf("instruction file became a task: %+v", r.Specs)
	}
}
func TestHistoricalCommitListBeatsSharedPaths(t *testing.T) {
	for _, withObject := range []bool{true, false} {
		t.Run(map[bool]string{true: "old commit object", false: "saved subject only"}[withObject], func(t *testing.T) {
			d := repo(t)
			write(t, d, "src/shared.go", "one")
			old := commit(t, d, "Implement editor contract")
			write(t, d, "src/shared.go", "two")
			command(t, d, "add", "src/shared.go")
			command(t, d, "commit", "--amend", "-qm", "Implement editor contract")
			head := command(t, d, "rev-parse", "HEAD")
			listed := "- `" + old[:9] + "` - Editor contract."
			if !withObject {
				listed = "- `aaaaaaaaa` `Implement editor contract`"
			}
			write(t, d, ".spec/phase1.md", "---\nstatus: done\n---\n`src/shared.go`\n### Commits\n"+listed+"\n")
			write(t, d, ".spec/umbrella.md", "---\nstatus: planning\n---\n`src/shared.go`\n")
			r := checked(t, d)
			if len(r.UnassignedCommits) != 0 || len(r.CommitAssociationIssues) != 0 {
				t.Fatalf("%+v", r)
			}
			for _, s := range r.Specs {
				if contains(s.Commits, head) != (s.Path == ".spec/phase1.md") {
					t.Fatalf("wrong owner: %+v", s)
				}
			}
		})
	}
}
func TestSharedTicketProducesExplainableIssueWithoutWarningSpam(t *testing.T) {
	d := repo(t)
	write(t, d, "src/a.go", "new feature")
	h := commit(t, d, "Implement later work\n\nPart of DEMO-12")
	for _, name := range []string{"highlevel", "phase1"} {
		write(t, d, ".spec/"+name+".md", "---\nstatus: planning\n---\n## Request\nTicket: DEMO-12\n")
	}
	r := checked(t, d)
	if !contains(r.UnassignedCommits, h) || len(r.CommitAssociationIssues) != 1 || len(r.Warnings) != 0 {
		t.Fatalf("%+v", r)
	}
	i := r.CommitAssociationIssues[0]
	if i.Subject != "Implement later work" || len(i.Candidates) != 2 || !strings.Contains(i.Reason, "ticket in commit message") {
		t.Fatalf("%+v", i)
	}
}
func TestCommitBodyTicketSelectsSpecAndRecordsTicketEvidence(t *testing.T) {
	d := repo(t)
	write(t, d, "src/a.go", "work")
	h := commit(t, d, "Implement feature\n\nPart of DEMO-42")
	write(t, d, ".spec/owner.md", "---\nstatus: planning\n---\n## Request\nTickets: DEMO-41, DEMO-42\n")
	write(t, d, ".spec/other.md", "---\nstatus: planning\n---\n## Request\nTicket: DEMO-99\n")
	r := checked(t, d)
	for _, s := range r.Specs {
		if s.Path != ".spec/owner.md" {
			continue
		}
		if !contains(s.Commits, h) {
			t.Fatal("body ticket did not associate commit")
		}
		for _, e := range s.Evidence {
			if e.Source == h && e.Kind == "commit message and spec share ticket DEMO-42" {
				return
			}
		}
		t.Fatal("ticket evidence missing")
	}
	t.Fatal("owner spec missing")
}
func TestUntrackedSpecOwnsCommitsAndDirtyCode(t *testing.T) {
	d := repo(t)
	write(t, d, "src/a.go", "one")
	h := commit(t, d, "implement")
	write(t, d, ".spec/current.md", "---\nstatus: executing subtask 2\n---\n## Request\nTicket: DEMO-12\n\nChanges `src/a.go`.\n")
	write(t, d, "src/a.go", "two")
	write(t, d, "_ai/MISTAKES.md", "lesson")
	r := checked(t, d)
	if len(r.Specs) != 1 {
		t.Fatalf("%+v", r)
	}
	s := r.Specs[0]
	if !contains(s.Commits, h) || !contains(s.DirtyFiles, "src/a.go") || s.Stage != "execution" || !contains(s.Tickets, "DEMO-12") {
		t.Fatalf("%+v", s)
	}
	if len(r.AuxiliaryFiles) != 1 {
		t.Fatal(r.AuxiliaryFiles)
	}
}
func TestPushedHistoryExcludedAcrossRemotes(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/done.md", "---\nstatus: done\n---\n")
	h := commit(t, d, "finished")
	command(t, d, "update-ref", "refs/remotes/other/feature", h)
	r := checked(t, d)
	if len(r.Commits) != 0 || len(r.Specs) != 0 {
		t.Fatalf("%+v", r)
	}
}
func TestNoRemoteDoesNotInventUnpushedHistory(t *testing.T) {
	d := repo(t)
	command(t, d, "update-ref", "-d", "refs/remotes/origin/main")
	write(t, d, ".spec/plan.md", "---\nstatus: planning\n---\n")
	r := checked(t, d)
	if len(r.Commits) != 0 || len(r.Specs) != 1 || len(r.Warnings) == 0 {
		t.Fatalf("%+v", r)
	}
}
func TestAmbiguousDirtyWorkStaysUnassigned(t *testing.T) {
	d := repo(t)
	for _, name := range []string{"a", "b"} {
		write(t, d, ".spec/"+name+".md", "---\nstatus: planning\n---\n`src/shared.go`\n")
	}
	write(t, d, "src/shared.go", "work")
	r := checked(t, d)
	if len(r.UnassignedFiles) != 1 {
		t.Fatalf("%+v", r)
	}
	for _, s := range r.Specs {
		if len(s.DirtyFiles) != 0 {
			t.Fatal(s)
		}
	}
}
func TestOrderingRecentPlanOldPlanDoneAndAuxiliary(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/done.md", "---\nstatus: done\n---\n")
	commit(t, d, "complete")
	write(t, d, ".spec/old.md", "---\nstatus: planning\n---\n")
	old := time.Now().Add(-30 * 24 * time.Hour)
	os.Chtimes(filepath.Join(d, ".spec/old.md"), old, old)
	write(t, d, ".spec/new.md", "---\nstatus: planning\n---\n")
	write(t, d, ".ai-mistakes.md", "fresh")
	r := checked(t, d)
	if len(r.Specs) != 3 || r.Specs[0].Path != ".spec/new.md" || r.Specs[1].Path != ".spec/old.md" || r.Specs[2].Path != ".spec/done.md" {
		t.Fatalf("%+v", r.Specs)
	}
}
func TestUnchangedPushedSpecCanOwnDirtyWork(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/work.md", "---\nstatus: executing\n---\n`src/feature.go`\n")
	h := commit(t, d, "plan")
	command(t, d, "update-ref", "refs/remotes/origin/main", h)
	write(t, d, "src/feature.go", "work")
	r := checked(t, d)
	if len(r.Specs) != 1 || len(r.Specs[0].DirtyFiles) != 1 {
		t.Fatalf("%+v", r)
	}
}
func TestDeletedSpecAndRenameWithSpaces(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/old name.md", "---\nstatus: planning\n---\n")
	h := commit(t, d, "plan")
	command(t, d, "update-ref", "refs/remotes/origin/main", h)
	command(t, d, "mv", ".spec/old name.md", ".spec/new name.md")
	r := checked(t, d)
	if len(r.Specs) != 2 {
		t.Fatalf("%+v", r)
	}
	deleted := false
	for _, s := range r.Specs {
		deleted = deleted || s.Deleted
	}
	if !deleted {
		t.Fatal("missing rename source")
	}
}
func TestMetadataDoesNotClaimReferencedTicketsOrCompletion(t *testing.T) {
	s := parseSpec(".spec/demo-10.md", "---\nstatus: completed subtask 24\n---\n## User request\nFix DEMO-10 and DEMO-11.\n\nBackground: DEMO-99.\n\nAlso include DEMO-12.\n## History\nDEMO-200\n")
	if strings.Join(s.Tickets, ",") != "DEMO-10,DEMO-11,DEMO-12" || s.Stage != "execution" {
		t.Fatalf("%+v", s)
	}
	if !contains(s.MentionedTickets, "DEMO-99") {
		t.Fatal(s)
	}
}
func TestMissingCheckoutIsAnError(t *testing.T) {
	r := scan(filepath.Join(t.TempDir(), "missing"), time.Now())
	if r.Error == "" {
		t.Fatal("missing error")
	}
}

func TestCommitOrderAndUntrackedStatus(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/a.md", "---\nstatus: done\n---\n")
	commit(t, d, "first")
	write(t, d, ".spec/z.md", "---\nstatus: done\n---\n")
	commit(t, d, "second")
	write(t, d, ".spec/plan.md", "---\nstatus: planning\n---\n")
	r := checked(t, d)
	if len(r.Specs) != 3 || r.Specs[0].GitStatus != "??" || r.Specs[1].Path != ".spec/z.md" || r.Specs[2].Path != ".spec/a.md" {
		t.Fatalf("unexpected order: %+v", r.Specs)
	}
}
