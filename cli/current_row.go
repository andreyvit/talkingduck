package cli

// History supplements current work; it never replaces the worktree summary.
func needsCurrentSummary(repo worktreeView) bool {
	if repo.HideSummary {
		return false
	}
	for _, task := range repo.Specs {
		if task.Workflow.Stage != "PUSHED" {
			return false
		}
	}
	return true
}
