//go:build darwin && cgo

package main

import (
	"context"
	"talkingduck/cli"
	"talkingduck/inventory"
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
