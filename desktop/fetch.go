//go:build darwin && cgo

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
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
		if ctx.Err() != nil || (v.cache != nil && !slices.Contains(v.cache.Paths(), path)) {
			return
		}
		err := v.fetch(ctx, path)
		if ctx.Err() != nil {
			return
		}
		v.recordFetchResult(path, err)
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

// The latest completed fetch decides connectivity; old folder failures do not
// keep the offline banner alive after another remote responds.
func (v *inventoryView) recordFetchResult(path string, err error) {
	v.fetchMu.Lock()
	v.noInternet = networkFetchFailure(err)
	if v.fetchErrors == nil {
		v.fetchErrors = make(map[string]string)
	}
	if err != nil && !v.noInternet {
		v.fetchErrors[path] = "fetch failed; remote counts may be stale: " + err.Error()
	} else {
		delete(v.fetchErrors, path)
	}
	v.fetchMu.Unlock()
	v.publish()
}

func networkFetchFailure(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(err.Error())
	// --all can report independent failures. Never hide an auth/repository error.
	for _, other := range []string{"permission denied", "authentication failed", "repository not found", "host key verification failed", "couldn't find remote ref", "cannot lock ref", "ssl certificate problem", "certificate verification failed", "returned error: 4", "returned error: 5"} {
		if strings.Contains(message, other) {
			return false
		}
	}
	for _, network := range []string{"network is unreachable", "network is down", "no route to host", "could not resolve host", "could not resolve hostname", "could not resolve proxy", "temporary failure in name resolution", "name or service not known", "nodename nor servname provided", "operation timed out", "connection timed out", "connection timeout", "connection reset", "connection refused", "connection aborted", "failed to connect", "couldn't connect", "could not connect", "unable to connect", "network connection was lost", "empty reply from server", "unexpected disconnect while reading sideband packet"} {
		if strings.Contains(message, network) {
			return true
		}
	}
	return false
}
