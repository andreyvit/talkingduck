package cli

// History supplements current work; it never replaces the worktree summary.
func needsCurrentSummary(repo worktreeView) bool {
	for _, task := range repo.Specs {
		if task.Workflow.Stage != "PUSHED" {
			return false
		}
	}
	return true
}
