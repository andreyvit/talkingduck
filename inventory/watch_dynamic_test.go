package inventory

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestConfiguredMembershipUsesNotifications(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changed := make(chan struct{}, 1)
	calls := make(chan struct{}, 10)
	var count atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		WatchConfiguredChanges(ctx, func() []string { count.Add(1); calls <- struct{}{}; return nil }, false, func(ChangeEvent) {}, func(error) {}, changed)
	}()
	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatal("no initial membership")
	}
	select {
	case <-calls:
		t.Fatal("polled membership without change")
	case <-time.After(1100 * time.Millisecond):
	}
	changed <- struct{}{}
	select {
	case <-calls:
	case <-time.After(time.Second):
		t.Fatal("membership change not read")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not stop")
	}
	if count.Load() != 2 {
		t.Fatal(count.Load())
	}
}
