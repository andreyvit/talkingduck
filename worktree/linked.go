package worktree

import (
	"os"
	"path/filepath"
	"strings"
)

// Linked lists live worktrees registered with this checkout's main repository.
// Git metadata supplies paths; no recursive disk scan or Git process is needed.
func Linked(root string) []string {
	if !IsGit(root) {
		return nil
	}
	main := Parent(root)
	if main == "" {
		main = root
	}
	if real, err := filepath.EvalSymlinks(main); err == nil {
		main = real
	}
	registry := filepath.Join(main, ".git", "worktrees")
	entries, _ := os.ReadDir(registry)
	var result []string
	for _, entry := range entries {
		b, err := os.ReadFile(filepath.Join(registry, entry.Name(), "gitdir"))
		if err != nil {
			continue
		}
		path := strings.TrimSpace(string(b))
		if !filepath.IsAbs(path) || filepath.Base(path) != ".git" {
			continue
		}
		path = filepath.Dir(path)
		if real, err := filepath.EvalSymlinks(path); err == nil {
			path = real
		}
		// Ignore stale registrations and entries that now point at another repo.
		if Parent(path) == main && IsGit(path) {
			result = append(result, path)
		}
	}
	return result
}
