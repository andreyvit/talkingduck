package worktree

import "testing"

func TestDisplayName(t *testing.T) {
	for _, tt := range []struct{ path, main, want string }{
		{"/home/user/.codex/worktrees/c9e4/backend", "/src/backend", "c9e4"},
		{"/custom/codex/worktrees/named/backend/backend/", "/src/backend", "named"},
		{"/src/backend/.claude/worktrees/feature-auth", "/src/backend", "feature-auth"},
		{"/custom/claude/backend-id/feature-auth", "/src/backend", "feature-auth"},
		{"/home/user/.grok/worktrees/backend/worktree-abc", "/src/backend", "worktree-abc"},
		{"/home/user/.grok/worktrees/task/backend", "/src/backend", "task"},
		{"/custom/feature/backend", "/src/backend", "feature"},
		{"/src/back11", "/src/backend", "back11"},
		{"/src/backend", "", "backend"},
		{"/src/backend", "/src/backend", "backend"},
		{"/other/backend", "", "backend"},
		{"/src/backend/.claude/worktrees/backend", "/src/backend", "backend"},
	} {
		if got := DisplayName(tt.path, tt.main); got != tt.want {
			t.Errorf("%s: got %q want %q", tt.path, got, tt.want)
		}
	}
}
