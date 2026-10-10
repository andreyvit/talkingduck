package worktree

import (
	"os"
	"path/filepath"
	"strings"
)

// Parent returns a linked worktree's original checkout, using Git's metadata.
// Directory nesting alone is not evidence of a parent checkout.
func Parent(root string) string {
	b, err := os.ReadFile(filepath.Join(root, ".git"))
	if err != nil {
		return ""
	}
	target, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir: ")
	if !ok {
		return ""
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	b, err = os.ReadFile(filepath.Join(target, "commondir"))
	if err != nil {
		return ""
	}
	common := strings.TrimSpace(string(b))
	if !filepath.IsAbs(common) {
		common = filepath.Join(target, common)
	}
	common = filepath.Clean(common)
	if filepath.Base(common) != ".git" {
		return ""
	}
	parent := filepath.Dir(common)
	if real, err := filepath.EvalSymlinks(parent); err == nil {
		parent = real
	}
	if parent == filepath.Clean(root) || !IsGit(parent) {
		return ""
	}
	return parent
}
