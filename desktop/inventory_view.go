//go:build darwin && cgo

package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"talkingduck/cli"
	"talkingduck/inventory"
	"talkingduck/monitor"
	"talkingduck/settings"
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
	folderOverride  bool
	folderMu        sync.Mutex
	settings        *settings.Store
	markdown        *markdownViewer
	settingsChanged func()
	settingsError   func(error)
	publicationMu   sync.Mutex
	generation      float64
	fetchSlots      chan struct{}
	fetchMu         sync.Mutex
	fetch           func(context.Context, string) error
	lastFetch       map[string]time.Time
	fetchErrors     map[string]string
	scanMu          sync.Mutex
	mu              sync.Mutex
	cache           inventoryCache
	state           inventoryState
	cheap           *monitor.Queue
	expensive       *monitor.Queue
	emit            func(inventoryState)
	updateTray      func(*inventory.Snapshot)
}

func (v *inventoryView) publish() {
	v.mu.Lock()
	state := v.state
	v.mu.Unlock()
	v.emit(state)
}

// Manual refresh enters the same full-reconciliation path as recovery.
func (v *inventoryView) refresh() {
	v.event(inventory.ChangeEvent{Reconcile: true})
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
	v.publicationMu.Lock()
	defer v.publicationMu.Unlock()
	generation, _ := snapshot.Overview["generation"].(float64)
	if generation > 0 && generation < v.generation {
		return
	}
	v.generation = generation
	v.fetchMu.Lock()
	for _, path := range v.cache.Paths() {
		if warning := v.fetchErrors[path]; warning != "" {
			table.Warnings = append(table.Warnings, filepath.Base(path)+": "+warning)
		}
	}
	v.fetchMu.Unlock()
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
	valid := false
	for _, p := range v.cache.Paths() {
		if p == path {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("worktree is not configured: %s", path)
	}
	if v.expensive != nil {
		return v.expensive.Request(ctx, path)
	}
	return v.scanFolder(ctx, path)
}

func (v *inventoryView) scanFolder(ctx context.Context, path string) error {
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
