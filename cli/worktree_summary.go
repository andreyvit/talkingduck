package cli

import "fmt"

// File cleanliness alone does not describe unassigned local work.
func worktreeSummary(repo worktreeView) string {
	if n := len(repo.UnassignedCommits); n > 0 {
		return fmt.Sprintf("%d unpushed commits; spec unknown", n)
	}
	return ""
}
