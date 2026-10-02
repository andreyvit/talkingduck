//go:build darwin && cgo

package main

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	"talkingduck/cli"
	"talkingduck/inventory"
)

type inventoryState struct {
	cli.TableView
	Loading        bool   `json:"loading"`
	Error          string `json:"error"`
	UpdatedAt      string `json:"updatedAt"`
	RefreshingPath string `json:"refreshingPath"`
}

type inventoryCache interface {
	Paths() []string
	Update(string) error
	Snapshot() *inventory.Snapshot
}

type inventoryView struct {
	scanMu     sync.Mutex
	mu         sync.Mutex
	cache      inventoryCache
	state      inventoryState
	requests   chan struct{}
	emit       func(inventoryState)
	updateTray func(*inventory.Snapshot)
}

func (v *inventoryView) publish() {
	v.mu.Lock()
	state := v.state
	v.mu.Unlock()
	v.emit(state)
}

// Manual refresh wakes a sleeping scheduler; requests during a pass coalesce.
func (v *inventoryView) refresh() {
	select {
	case v.requests <- struct{}{}:
	default:
	}
}

func (v *inventoryView) render(path string, complete bool) {
	snapshot := v.cache.Snapshot()
	table, err := cli.BuildTable(snapshot.Overview)
	for _, repo := range snapshot.Details.Repositories {
		if repo.Cache != nil && repo.Cache.Error != "" {
			table.Warnings = append(table.Warnings, filepath.Base(repo.Path)+": refresh failed: "+repo.Cache.Error)
		}
	}
	v.mu.Lock()
	v.state.Loading = !complete
	v.state.RefreshingPath = ""
	if path != "" {
		v.state.RefreshingPath = filepath.Base(path)
	}
	if complete {
		v.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	v.mu.Unlock()
	v.publishTable(snapshot, table, err)
}

func (v *inventoryView) publishTable(snapshot *inventory.Snapshot, table cli.TableView, err error) {
	v.mu.Lock()
	if err != nil {
		v.state.Error = err.Error()
	} else {
		v.state.Error = ""
		v.state.TableView = table
	}
	v.mu.Unlock()
	v.publish()
	if v.updateTray != nil {
		v.updateTray(snapshot)
	}
}

func (v *inventoryView) Paths() []string               { return v.cache.Paths() }
func (v *inventoryView) Snapshot() *inventory.Snapshot { return v.cache.Snapshot() }

// Share the scheduler's scan lock so MCP never overlaps a background scan.
func (v *inventoryView) RefreshFolder(ctx context.Context, path string) error {
	v.scanMu.Lock()
	defer v.scanMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	err := v.cache.Update(path)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	snapshot := v.cache.Snapshot()
	table, tableErr := cli.BuildTable(snapshot.Overview)
	for _, repo := range snapshot.Details.Repositories {
		if repo.Cache != nil && repo.Cache.Error != "" {
			table.Warnings = append(table.Warnings, filepath.Base(repo.Path)+": refresh failed: "+repo.Cache.Error)
		}
	}
	v.publishTable(snapshot, table, tableErr)
	if err != nil {
		return err
	}
	return tableErr
}

// One sequential pass, then a full interval of rest. Cancellation stops before
// the next folder; the collector bounds its current scan but cannot cancel it.
func (v *inventoryView) run(ctx context.Context, interval time.Duration) {
	for {
		for _, path := range v.cache.Paths() {
			if ctx.Err() != nil {
				return
			}
			v.scanMu.Lock()
			if ctx.Err() != nil {
				v.scanMu.Unlock()
				return
			}
			v.render(path, false)
			if err := v.cache.Update(path); err != nil {
				log.Printf("inventory: %s: %v", path, err)
			}
			if ctx.Err() != nil {
				v.scanMu.Unlock()
				return
			}
			v.render(path, false)
			v.scanMu.Unlock()
			log.Printf("inventory: refreshed %s", filepath.Base(path))
		}
		if ctx.Err() != nil {
			return
		}
		v.scanMu.Lock()
		v.render("", true)
		v.scanMu.Unlock()
		log.Printf("inventory: pass complete; next pass in %s", interval)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-v.requests:
			timer.Stop()
		case <-timer.C:
		}
	}
}
