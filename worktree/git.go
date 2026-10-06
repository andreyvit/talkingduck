// Package worktree identifies Git checkout roots through cheap filesystem checks.
package worktree

import (
	"os"
	"path/filepath"
	"strings"
)

// IsGit accepts ordinary checkouts, linked worktrees, and submodules. It excludes
// bare repositories and nested working directories; no Git process or scan runs.
func IsGit(path string) bool {
	gitdir := filepath.Join(path, ".git")
	info, err := os.Stat(gitdir)
	if err != nil {
		return false
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() || info.Size() > 4096 {
			return false
		}
		data, err := os.ReadFile(gitdir)
		if err != nil || !strings.HasPrefix(string(data), "gitdir: ") {
			return false
		}
		target := strings.TrimSpace(strings.TrimPrefix(string(data), "gitdir: "))
		if target == "" || strings.ContainsAny(target, "\r\n") {
			return false
		}
		gitdir = target
		if !filepath.IsAbs(target) {
			gitdir = filepath.Join(path, target)
		}
	}
	head, err := os.Stat(filepath.Join(gitdir, "HEAD"))
	if err != nil || !head.Mode().IsRegular() {
		return false
	}
	common := gitdir
	if data, err := os.ReadFile(filepath.Join(gitdir, "commondir")); err == nil {
		common = strings.TrimSpace(string(data))
		if !filepath.IsAbs(common) {
			common = filepath.Join(gitdir, common)
		}
	}
	objects, err := os.Stat(filepath.Join(common, "objects"))
	return err == nil && objects.IsDir()
}
