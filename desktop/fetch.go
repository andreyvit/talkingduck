//go:build darwin && cgo

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func fetchRemote(ctx context.Context, path string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	return retryFetchLocks(ctx, 250*time.Millisecond, func() error {
		cmd := exec.CommandContext(ctx, "git", "-C", path, "-c", "credential.interactive=false", "fetch", "--all", "--quiet")
		cmd.WaitDelay = 2 * time.Second
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=15")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	})
}

// Record attempts before launch; scans never wait for network access.
func (v *inventoryView) fetchIfDue(ctx context.Context, path string) {
	if v.fetch == nil || ctx.Err() != nil {
		return
	}
	v.fetchMu.Lock()
	defer v.fetchMu.Unlock()
	if v.lastFetch == nil {
		v.lastFetch = make(map[string]time.Time)
		v.fetchErrors = make(map[string]string)
	}
	if last, ok := v.lastFetch[path]; ok && time.Since(last) < 15*time.Minute {
		return
	}
	if v.fetchSlots == nil {
		v.fetchSlots = make(chan struct{}, 5)
	}
	slots := v.fetchSlots
	v.lastFetch[path] = time.Now()
	go func() {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-slots }()
		if ctx.Err() != nil {
			return
		}
		err := v.fetch(ctx, path)
		v.fetchMu.Lock()
		defer v.fetchMu.Unlock()
		if err != nil {
			v.fetchErrors[path] = "fetch failed; remote counts may be stale: " + err.Error()
		} else {
			delete(v.fetchErrors, path)
		}
	}()
}

// Three retries after the initial attempt, within the same fetch slot and timeout.
func retryFetchLocks(ctx context.Context, delay time.Duration, attempt func() error) error {
	for retry := 0; ; retry++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := attempt()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !onlyRefLockFailure(err.Error()) {
			return err
		}
		if retry == 3 {
			return nil
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// Preserve independent errors when --all reports failures from multiple remotes.
func onlyRefLockFailure(message string) bool {
	message = strings.ToLower(message)
	if !strings.Contains(message, "cannot lock ref") {
		return false
	}
	for _, line := range strings.Split(message, "\n") {
		if (strings.Contains(line, "fatal:") || strings.Contains(line, "error:")) &&
			!strings.Contains(line, "cannot lock ref") && !strings.Contains(line, "could not fetch ") {
			return false
		}
	}
	return true
}
