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
