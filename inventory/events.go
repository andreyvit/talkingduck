package inventory

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fswatcher/fswatcher"
)

// ChangeEvent is a hint, never a replacement for collection. Empty Path with
// Reconcile requests all folders; Providers requests scoped provider discovery.
type ChangeEvent struct {
	Path                                   string
	Cheap, Expensive, Providers, Reconcile bool
}

func realPath(path string) string {
	p, e := filepath.Abs(path)
	if e != nil {
		return path
	}
	if r, e := filepath.EvalSymlinks(p); e == nil {
		return r
	}
	return p
}
func under(path, root string) bool {
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}
func providerRoots(home string) []string {
	return []string{
		envPath("CODEX_HOME", filepath.Join(home, ".codex")),
		envPath("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude")),
		envPath("GROK_HOME", filepath.Join(home, ".grok")),
		filepath.Join(home, "Library", "Application Support", "Claude", "claude-code-sessions"),
	}
}
func providerFile(path string) bool {
	name := filepath.Base(path)
	return strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".sqlite") || strings.HasSuffix(name, ".sqlite-wal")
}

// WatchChanges registers native recursive watches before requesting startup
// reconciliation. Recovery shares that same event path and repairs registrations.
// It does no collection or throttling; the consumer owns both queues.
func WatchChanges(ctx context.Context, paths []string, home string, agents bool, recovery time.Duration, emit func(ChangeEvent), warn func(error)) {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if recovery <= 0 {
		recovery = 10 * time.Minute
	}
	w, err := fswatcher.NewWatcher()
	if err != nil {
		warn(err)
		fallbackReconciliation(ctx, emit)
		return
	}
	defer w.Close()
	roots := map[string][]string{}
	providers := []string{}
	registered := map[string]bool{}
	if agents {
		for _, p := range providerRoots(home) {
			providers = append(providers, realPath(p))
		}
	}
	register := func(path string) {
		// Watch the nearest existing parent when a provider or checkout is absent.
		// Event routing still filters against the intended root.
		for {
			st, e := os.Stat(path)
			if e == nil && st.IsDir() {
				break
			}
			parent := filepath.Dir(path)
			if parent == path {
				return
			}
			path = parent
		}
		for p := range registered {
			if under(path, p) {
				return
			}
		}
		if e := w.AddRecursive(path, fswatcher.All); e != nil {
			warn(fmt.Errorf("watch %s: %w", path, e))
			return
		}
		registered[path] = true
	}
	refill := func() {
		for _, path := range paths {
			root := realPath(path)
			roots[root] = unique(append(roots[root], path))
			register(root)
			for _, p := range gitWatchRoots(path) {
				p = realPath(p)
				roots[p] = unique(append(roots[p], path))
				register(p)
			}
		}
		for _, p := range providers {
			register(p)
		}
		emit(ChangeEvent{Reconcile: true})
	}
	refill()
	ticker := time.NewTicker(recovery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Recreate registrations too: root replacement can invalidate a native watch.
			for p := range registered {
				_ = w.Remove(p)
				delete(registered, p)
			}
			roots = map[string][]string{}
			refill()
		case e, ok := <-w.Errors:
			if !ok {
				warn(fmt.Errorf("filesystem error stream closed"))
				fallbackReconciliation(ctx, emit)
				return
			}
			warn(e)
			emit(ChangeEvent{Reconcile: true})
		case ev, ok := <-w.Events:
			if !ok {
				warn(fmt.Errorf("filesystem event stream closed"))
				fallbackReconciliation(ctx, emit)
				return
			}
			path := filepath.Clean(ev.Name)
			for root, folders := range roots {
				if under(path, root) {
					for _, folder := range folders {
						cheap := under(path, filepath.Join(realPath(folder), ".spec"))
						emit(ChangeEvent{Path: folder, Cheap: cheap, Expensive: true})
					}
				}
			}
			for _, root := range providers {
				if !strings.HasSuffix(path, ".sqlite-shm") && under(path, root) && (providerFile(path) || ev.Op.Has(fswatcher.Create|fswatcher.Remove|fswatcher.Rename)) {
					emit(ChangeEvent{Providers: true})
					break
				}
			}
		}
	}
}

// Resolve standard checkout/worktree Git indirections without launching Git.
func gitWatchRoots(root string) []string {
	dir := filepath.Join(root, ".git")
	if b, e := os.ReadFile(dir); e == nil {
		value, ok := strings.CutPrefix(strings.TrimSpace(string(b)), "gitdir:")
		if !ok {
			return nil
		}
		dir = strings.TrimSpace(value)
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, dir)
		}
	}
	result := []string{dir}
	if b, e := os.ReadFile(filepath.Join(dir, "commondir")); e == nil {
		common := strings.TrimSpace(string(b))
		if !filepath.IsAbs(common) {
			common = filepath.Join(dir, common)
		}
		result = append(result, filepath.Clean(common))
	}
	return result
}

func fallbackReconciliation(ctx context.Context, emit func(ChangeEvent)) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		emit(ChangeEvent{Reconcile: true})
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
