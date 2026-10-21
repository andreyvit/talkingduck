//go:build darwin && cgo

package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFetchRunsWithoutBlockingAndThrottlesAttempts(t *testing.T) {
	started := make(chan string, 3)
	release := make(chan struct{})
	defer close(release)
	v := &inventoryView{fetch: func(_ context.Context, path string) error { started <- path; <-release; return errors.New("offline") }}
	ctx := context.Background()
	v.fetchIfDue(ctx, "a")
	v.fetchIfDue(ctx, "a")
	v.fetchIfDue(ctx, "b")
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("fetch not launched")
		}
	}
	select {
	case <-started:
		t.Fatal("duplicate fetch")
	default:
	}
	v.fetchMu.Lock()
	if len(v.lastFetch) != 2 {
		t.Error(v.lastFetch)
	}
	v.fetchMu.Unlock()
}
func TestFailedFetchIsThrottled(t *testing.T) {
	v := &inventoryView{fetch: func(context.Context, string) error { t.Error("retried too soon"); return nil }, lastFetch: map[string]time.Time{"a": time.Now()}, fetchErrors: map[string]string{"a": "offline"}}
	v.fetchIfDue(context.Background(), "a")
}

func TestFetchConcurrencyLimit(t *testing.T) {
	started := make(chan string, 6)
	release := make(chan struct{})
	defer close(release)
	v := &inventoryView{fetch: func(_ context.Context, path string) error { started <- path; <-release; return nil }}
	for _, path := range []string{"a", "b", "c", "d", "e", "f"} {
		v.fetchIfDue(context.Background(), path)
	}
	for range 5 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("fetch not started")
		}
	}
	select {
	case <-started:
		t.Fatal("more than five concurrent fetches")
	case <-time.After(30 * time.Millisecond):
	}
	release <- struct{}{}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("queued fetch did not start")
	}
}

func TestRetryFetchLocks(t *testing.T) {
	lock := errors.New("error: cannot lock ref 'refs/remotes/origin/main'")
	other := errors.New("authentication failed")
	mixed := errors.New("error: cannot lock ref x\nfatal: authentication failed")
	for _, tc := range []struct {
		name    string
		results []error
		calls   int
		want    error
	}{
		{"recovers", []error{lock, nil}, 2, nil},
		{"exhausted locks ignored", []error{lock, lock, lock, lock}, 4, nil},
		{"other failure", []error{other}, 1, other},
		{"mixed failure preserved", []error{mixed}, 1, mixed},
		{"different failure after lock", []error{lock, other}, 2, other},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := retryFetchLocks(context.Background(), 0, func() error { result := tc.results[calls]; calls++; return result })
			if !errors.Is(err, tc.want) || calls != tc.calls {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := retryFetchLocks(ctx, time.Hour, func() error { calls++; cancel(); return lock })
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal(calls, err)
	}
}

func TestNetworkFetchFailure(t *testing.T) {
	for _, message := range []string{
		"exit status 128: ssh: connect to host github.com port 22: Operation timed out\nfatal: Could not read from remote repository.\nPlease make sure you have the correct access rights\nand the repository exists.",
		"fatal: unable to access 'https://github.com/example/repo': Could not resolve host: github.com",
		"ssh: Could not resolve hostname github.com: nodename nor servname provided, or not known",
		"Network is unreachable", "No route to host", "Failed to connect to github.com port 443", "Connection reset by peer",
	} {
		if !networkFetchFailure(errors.New(message)) {
			t.Error("not classified", message)
		}
	}
	for _, err := range []error{nil, context.Canceled, errors.New("Permission denied (publickey)"), errors.New("Repository not found"), errors.New("Could not read from remote repository"), errors.New("Operation timed out\nfatal: Authentication failed"), errors.New("SSL certificate problem"), errors.New("cannot lock ref 'origin/main'")} {
		if networkFetchFailure(err) {
			t.Error("hidden non-network error", err)
		}
	}
	if !networkFetchFailure(context.DeadlineExceeded) {
		t.Fatal("fetch deadline not classified")
	}
}
func TestFetchConnectivityStatusClearsOnAnyResponse(t *testing.T) {
	var states []inventoryState
	v := &inventoryView{emit: func(s inventoryState) { states = append(states, s) }}
	offline := errors.New("ssh: connect to host github.com port 22: Operation timed out")
	v.recordFetchResult("a", offline)
	v.recordFetchResult("b", offline)
	if !states[len(states)-1].NoInternet || len(v.fetchErrors) != 0 {
		t.Fatal("network failures were not collapsed")
	}
	v.recordFetchResult("c", nil)
	if states[len(states)-1].NoInternet {
		t.Fatal("success did not clear offline")
	}
	v.recordFetchResult("a", offline)
	v.recordFetchResult("b", errors.New("Permission denied (publickey)"))
	if states[len(states)-1].NoInternet || v.fetchErrors["b"] == "" {
		t.Fatal("other error not preserved")
	}
	v.recordFetchResult("b", nil)
	if len(v.fetchErrors) != 0 {
		t.Fatal("success retained warning")
	}
}

func TestFetchWarningsPublishWithoutAccumulating(t *testing.T) {
	var state inventoryState
	v := &inventoryView{cache: &testCache{}, emit: func(s inventoryState) { state = s }}
	v.recordFetchResult("/a", errors.New("Permission denied (publickey)"))
	v.publish()
	if len(state.Warnings) != 1 {
		t.Fatal("warning lost or duplicated", state.Warnings)
	}
	v.recordFetchResult("/a", errors.New("Network is unreachable"))
	if len(state.Warnings) != 0 || !state.NoInternet {
		t.Fatal("network error kept old warning", state)
	}
	v.recordFetchResult("/b", nil)
	if state.NoInternet || len(state.Warnings) != 0 {
		t.Fatal("success did not clear status", state)
	}
}
