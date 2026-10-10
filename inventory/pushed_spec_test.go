package inventory

import "testing"

func TestCompletedPushedSpecIsNotRevivedBySourceOverlap(t *testing.T) {
	for _, reference := range []string{"src/shared.go", "src/feature/"} {
		t.Run(reference, func(t *testing.T) {
			d := repo(t)
			source := "src/shared.go"
			if reference == "src/feature/" {
				source = "src/feature/new.go"
			}
			write(t, d, ".spec/old.md", "---\nstatus: done\n---\n`"+reference+"`\n")
			write(t, d, source, "old")
			h := commit(t, d, "complete old task")
			command(t, d, "update-ref", "refs/remotes/origin/main", h)
			write(t, d, source, "unrelated new work")
			r := checked(t, d)
			if len(r.Specs) != 0 || len(r.UnassignedFiles) != 1 {
				t.Fatalf("revived completed task: %+v", r)
			}
			// Explicitly changing the spec still reopens it, even before status is updated.
			write(t, d, ".spec/old.md", "---\nstatus: done\n---\n`"+reference+"`\nNew revision requested.\n")
			r = checked(t, d)
			if len(r.Specs) != 1 || !r.Specs[0].Dirty {
				t.Fatal("explicit local spec change lost")
			}
			commit(t, d, "reopen old task")
			r = checked(t, d)
			if len(r.Specs) != 1 || len(r.Specs[0].Commits) != 1 {
				t.Fatal("unpushed completed spec hidden")
			}
		})
	}
}

func TestHistoricalCandidateWithoutOwnedChangesIsDiscarded(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/old.md", "---\nstatus: executing\n---\n`src/shared.go`\n")
	write(t, d, "src/shared.go", "base")
	h := commit(t, d, "old task")
	command(t, d, "update-ref", "refs/remotes/origin/main", h)
	write(t, d, ".spec/current.md", "---\nstatus: executing\n---\n# Current task\n")
	write(t, d, "src/shared.go", "current task implementation")
	commit(t, d, "current implementation")
	write(t, d, "src/shared.go", "current dirty edit")
	r := checked(t, d)
	if len(r.Specs) != 1 || r.Specs[0].Path != ".spec/current.md" {
		t.Fatalf("ownerless historical candidate survived: %+v", r.Specs)
	}
}

func TestPushedProposalCannotBecomeLocalFromSourceMentions(t *testing.T) {
	d := repo(t)
	write(t, d, ".spec/proposal.md", "# Old research proposal\nNot changing code yet. Inventory `src/settings.go`.\n")
	write(t, d, "src/settings.go", "base")
	command(t, d, "update-ref", "refs/remotes/origin/main", commit(t, d, "proposal"))
	write(t, d, "src/settings.go", "unrelated setting change")
	r := checked(t, d)
	if len(r.Specs) != 0 || len(r.UnassignedFiles) != 1 {
		t.Fatalf("source edit revived untouched proposal: %+v", r.Specs)
	}
}
