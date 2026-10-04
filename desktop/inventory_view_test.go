//go:build darwin && cgo

package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"talkingduck/inventory"
)

type testCache struct {
	updates chan string
	release chan struct{}
}

func (c *testCache) Paths() []string { return []string{"/a", "/b"} }
func (c *testCache) Update(p string) error {
	c.updates <- p
	<-c.release
	return errors.New("synthetic failure")
}
func (c *testCache) Snapshot() *inventory.Snapshot {
	return &inventory.Snapshot{Overview: inventory.Object{"repositories": []any{}}}
}

func TestInventoryTrayUsesCacheAndFirstTask(t *testing.T) {
	snapshot := &inventory.Snapshot{Overview: inventory.Object{"repositories": []any{
		map[string]any{"cache": map[string]any{"state": "ready"}, "specs": []any{map[string]any{"workflow": map[string]any{"stage": "LL", "progress": "▶"}}, map[string]any{"workflow": map[string]any{"stage": "FIN", "progress": "✓"}}}},
		map[string]any{"cache": map[string]any{"state": "error"}, "git": map[string]any{"cleanliness": "CLEAN"}},
		map[string]any{"cache": map[string]any{"state": "ready"}, "git": map[string]any{"cleanliness": "CLEAN"}},
	}}}
	cells := inventoryCells(snapshot)
	if len(cells) != 3 || cells[0].Stage != "L" || cells[0].State != "running" || cells[1].State != "unknown" || cells[2].Stage != "-" {
		t.Fatal(cells)
	}
}

func TestTargetedRefreshWaitsForScannerAndPublishesFailure(t *testing.T) {
	c := &testCache{updates: make(chan string, 1), release: make(chan struct{}, 1)}
	published := make(chan inventoryState, 1)
	v := &inventoryView{cache: c, emit: func(s inventoryState) { published <- s }}
	v.scanMu.Lock()
	done := make(chan error, 1)
	go func() { done <- v.RefreshFolder(context.Background(), "/b") }()
	select {
	case <-c.updates:
		t.Fatal("overlapped scanner")
	case <-time.After(20 * time.Millisecond):
	}
	v.scanMu.Unlock()
	select {
	case path := <-c.updates:
		if path != "/b" {
			t.Fatal(path)
		}
	case <-time.After(time.Second):
		t.Fatal("no targeted scan")
	}
	c.release <- struct{}{}
	if err := <-done; err == nil {
		t.Fatal("lost refresh error")
	}
	select {
	case <-published:
	case <-time.After(time.Second):
		t.Fatal("did not publish cache after failed scan")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := v.RefreshFolder(ctx, "/a"); err != context.Canceled {
		t.Fatal(err)
	}
	select {
	case <-c.updates:
		t.Fatal("scanned cancelled request")
	default:
	}
}
