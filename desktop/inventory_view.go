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
	v.state.RefreshingPath = filepath.Base(path)
	if path == "" {
		v.state.RefreshingPath = ""
	}
	if err != nil {
		v.state.Error = err.Error()
	} else {
		v.state.Error = ""
		v.state.TableView = table
	}
	if complete {
		v.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	v.mu.Unlock()
	v.publish()
	if v.updateTray != nil {
		v.updateTray(snapshot)
	}
}

// One sequential pass, then a full interval of rest. Cancellation stops before
// the next folder; the collector bounds its current scan but cannot cancel it.
func (v *inventoryView) run(ctx context.Context, interval time.Duration) {
	for {
		for _, path := range v.cache.Paths() {
			if ctx.Err() != nil {
				return
			}
			v.render(path, false)
			if err := v.cache.Update(path); err != nil {
				log.Printf("inventory: %s: %v", path, err)
			}
			if ctx.Err() != nil {
				return
			}
			v.render(path, false)
			log.Printf("inventory: refreshed %s", filepath.Base(path))
		}
		if ctx.Err() != nil {
			return
		}
		v.render("", true)
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
