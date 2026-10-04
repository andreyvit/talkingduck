//go:build darwin && cgo

package main

import (
	"context"
	"talkingduck/cli"
	"talkingduck/inventory"
	"talkingduck/monitor"
	"testing"
	"time"
)

type liveTestCache struct{ calls chan string }

func (c *liveTestCache) Paths() []string                   { return []string{"/a", "/b"} }
func (c *liveTestCache) Update(string) error               { panic("cheap refresh called full scanner") }
func (c *liveTestCache) UpdateLive(p string) (bool, error) { c.calls <- p; return true, nil }
func (c *liveTestCache) Snapshot() *inventory.Snapshot {
	return &inventory.Snapshot{Overview: inventory.Object{"repositories": []any{}}}
}
func TestLiveRefreshDoesNotWaitForFullScanLock(t *testing.T) {
	c := &liveTestCache{calls: make(chan string, 2)}
	v := &inventoryView{cache: c, emit: func(inventoryState) {}}
	v.scanMu.Lock()
	defer v.scanMu.Unlock()
	done := make(chan struct{})
	go func() { defer close(done); v.refreshChanged(context.Background(), "/b") }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("live refresh waited for full scan")
	}
	if got := <-c.calls; got != "/b" {
		t.Fatal(got)
	}
}

func TestOlderPublicationCannotOverwriteLiveTable(t *testing.T) {
	c := &liveTestCache{calls: make(chan string, 1)}
	v := &inventoryView{cache: c, emit: func(inventoryState) {}}
	newer := &inventory.Snapshot{Overview: inventory.Object{"generation": float64(2)}}
	older := &inventory.Snapshot{Overview: inventory.Object{"generation": float64(1)}}
	v.publishTable(newer, cli.TableView{Rows: [][]string{{"new"}}}, nil)
	v.publishTable(older, cli.TableView{Rows: [][]string{{"old"}}}, nil)
	if v.state.Rows[0][0] != "new" {
		t.Fatal(v.state.Rows)
	}
}

func TestSourceEventsAndManualRefreshShareQueues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cheap, full := make(chan string, 8), make(chan string, 8)
	v := &inventoryView{cache: &liveTestCache{}, cheap: monitor.NewQueue(monitor.Policy{}), expensive: monitor.NewQueue(monitor.Policy{})}
	go v.cheap.Run(ctx, func(_ context.Context, path string) error { cheap <- path; return nil })
	go v.expensive.Run(ctx, func(_ context.Context, path string) error { full <- path; return nil })
	receive := func(ch chan string, want string) {
		t.Helper()
		select {
		case got := <-ch:
			if got != want {
				t.Fatalf("%q != %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatal("missing queued job")
		}
	}
	v.event(inventory.ChangeEvent{Path: "/b", Cheap: true, Expensive: true})
	receive(cheap, "/b")
	receive(full, "/b")
	// Process events use this same route; no direct cache update.
	v.event(inventory.ChangeEvent{Path: "/a", Expensive: true})
	receive(full, "/a")
	done := make(chan error, 1)
	go func() { done <- v.RefreshFolder(ctx, "/b") }()
	receive(full, "/b")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	v.event(inventory.ChangeEvent{Providers: true})
	receive(cheap, providerQueueKey)
	v.refresh()
	receive(cheap, providerQueueKey)
	receive(cheap, "/a")
	receive(cheap, "/b")
	receive(full, "/a")
	receive(full, "/b")
}
