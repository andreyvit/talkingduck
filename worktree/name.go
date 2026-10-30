package worktree

import "path/filepath"

// DisplayName removes redundant repository-name wrappers from linked worktrees.
// It uses the known main checkout, not assumptions about an agent's home path.
func DisplayName(path, main string) string {
	path = filepath.Clean(path)
	fallback := filepath.Base(path)
	if main == "" || path == filepath.Clean(main) {
		return fallback
	}
	base := filepath.Base(filepath.Clean(main))
	for filepath.Base(path) == base {
		parent := filepath.Dir(path)
		if parent == path || parent == filepath.Clean(main) {
			return fallback
		}
		path = parent
	}
	name := filepath.Base(path)
	// A repeated name with no distinguishing component is better than a label
	// naming the storage container (for example .claude/worktrees/backend).
	if name == "." || name == string(filepath.Separator) || name == "worktrees" || name == ".claude-worktrees" {
		return fallback
	}
	return name
}
