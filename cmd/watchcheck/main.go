// watchcheck tests native recursive notifications without changing existing files.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/fswatcher/fswatcher"
)

func main() {
	probe := flag.Bool("probe", false, "create/edit/rename/delete temporary files in each watched folder")
	timeout := flag.Duration("timeout", 5*time.Second, "maximum wait for each probe event")
	flag.Parse()
	if !*probe || flag.NArg() == 0 || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "Usage: watchcheck -probe [-timeout 5s] /path/to/worktree ...")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := check(ctx, flag.Args(), *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func check(ctx context.Context, paths []string, timeout time.Duration) error {
	w, err := fswatcher.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	roots := []string{}
	for _, p := range paths {
		root, err := filepath.Abs(p)
		if err == nil {
			root, err = filepath.EvalSymlinks(root)
		}
		if err != nil {
			return err
		}
		st, err := os.Stat(root)
		if err != nil {
			return err
		}
		if !st.IsDir() {
			return fmt.Errorf("not a directory: %s", root)
		}
		if err = w.AddRecursive(root, fswatcher.All); err != nil {
			return fmt.Errorf("watch %s: %w", root, err)
		}
		roots = append(roots, root)
	}
	failures := 0
	for _, root := range roots {
		if err := probeRoot(ctx, w, root, timeout); err != nil {
			fmt.Printf("FAIL %s: %v\n", root, err)
			failures++
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	fmt.Printf("%d/%d worktrees passed\n", len(roots)-failures, len(roots))
	if failures > 0 {
		return fmt.Errorf("%d worktrees missed probe events", failures)
	}
	return nil
}

// Drain notifications from the previous step before issuing the next operation.
// FileEvents may combine flags; this checks delivery, not an exact event count.
func settle(ctx context.Context, w *fswatcher.Watcher) error {
	timer := time.NewTimer(150 * time.Millisecond)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		case _, ok := <-w.Events:
			if !ok {
				return fmt.Errorf("event stream closed")
			}
		case err, ok := <-w.Errors:
			if !ok {
				return fmt.Errorf("error stream closed")
			}
			return err
		}
	}
}
func await(ctx context.Context, w *fswatcher.Watcher, path string, op fswatcher.Op, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("no %s event for %s within %s", op, path, timeout)
		case err, ok := <-w.Errors:
			if !ok {
				return fmt.Errorf("error stream closed")
			}
			return err
		case ev, ok := <-w.Events:
			if !ok {
				return fmt.Errorf("event stream closed")
			}
			if ev.Name == path && ev.Op&op != 0 {
				return nil
			}
		}
	}
}
func probeRoot(ctx context.Context, w *fswatcher.Watcher, root string, timeout time.Duration) (result error) {
	// MkdirTemp owns a unique subtree; cleanup never removes user files elsewhere.
	dir, err := os.MkdirTemp(root, ".talkingduck-watchcheck-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			result = errors.Join(result, fmt.Errorf("remove probe %s: %w", dir, err))
		}
	}()
	if err = await(ctx, w, dir, fswatcher.Create, timeout); err != nil {
		return err
	}
	nested := filepath.Join(dir, "nested")
	type probeStep struct {
		name, path string
		op         fswatcher.Op
		run        func() error
	}
	steps := []probeStep{
		{"nested mkdir", nested, fswatcher.Create, func() error { return os.Mkdir(nested, 0700) }},
	}
	for _, base := range []string{dir, nested} {
		p := filepath.Join(base, "probe.txt")
		renamed := filepath.Join(base, "renamed.txt")
		steps = append(steps,
			probeStep{"create", p, fswatcher.Create, func() error { return os.WriteFile(p, []byte("first\n"), 0600) }},
			probeStep{"edit", p, fswatcher.Write, func() error { return os.WriteFile(p, []byte("second\n"), 0600) }},
			probeStep{"rename", p, fswatcher.Rename, func() error { return os.Rename(p, renamed) }},
			probeStep{"delete", renamed, fswatcher.Remove, func() error { return os.Remove(renamed) }},
		)
	}
	var slowest time.Duration
	for _, step := range steps {
		if err = settle(ctx, w); err != nil {
			return err
		}
		start := time.Now()
		if err = step.run(); err != nil {
			return err
		}
		if err = await(ctx, w, step.path, step.op, timeout); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
		slowest = max(slowest, time.Since(start))
	}
	fmt.Printf("PASS %s: create/edit/rename/delete, nested + new directories; max %s\n", root, slowest.Round(time.Millisecond))
	return nil
}
