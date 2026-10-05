//go:build darwin && cgo

package main

import (
	"context"
	"log"
	"slices"
	"time"

	"talkingduck/cli"
	"talkingduck/inventory"
	"talkingduck/monitor"
)

const providerQueueKey = "@providers"

// Initialize before exposing MCP or UI refresh actions; queue pointers then stay
// immutable. Workers run until app cancellation; cheap and full work never share
// a worker or collection lock.
func (v *inventoryView) startQueues(ctx context.Context, agents bool) {
	v.cheap = monitor.NewQueue(monitor.Cheap)
	v.expensive = monitor.NewQueue(monitor.Expensive)
	probePaths := v.cache.Paths()
	probe := inventory.NewChangeProbe(probePaths, "", agents)
	go v.cheap.Run(ctx, func(ctx context.Context, path string) error {
		if path == providerQueueKey {
			next := v.cache.Paths()
			if !slices.Equal(next, probePaths) {
				probePaths = next
				probe = inventory.NewChangeProbe(next, "", agents)
			}
			changed, warnings := probe.Poll()
			for _, warning := range warnings {
				log.Printf("inventory change signals: %s", warning)
			}
			for _, folder := range changed {
				v.event(inventory.ChangeEvent{Path: folder, Cheap: true, Expensive: true})
			}
		} else {
			v.refreshChanged(ctx, path)
		}
		return nil
	})
	go v.expensive.Run(ctx, func(ctx context.Context, path string) error {
		if !slices.Contains(v.cache.Paths(), path) {
			return nil
		}
		v.render(path, false)
		started := time.Now()
		err := v.scanFolder(ctx, path)
		if err != nil {
			log.Printf("inventory: %s: %v", path, err)
		} else {
			log.Printf("inventory: refreshed %s in %s", path, time.Since(started).Round(time.Millisecond))
		}
		if ctx.Err() == nil {
			v.render("", true)
		}
		return err
	})
}

func (v *inventoryView) event(e inventory.ChangeEvent) {
	if v.cheap == nil || v.expensive == nil {
		return
	}
	if e.Reconcile {
		v.cheap.Notify(providerQueueKey)
		for _, path := range v.cache.Paths() {
			v.cheap.Notify(path)
			v.expensive.Notify(path)
		}
		return
	}
	if e.Providers {
		v.cheap.Notify(providerQueueKey)
	}
	if e.Path != "" {
		if e.Cheap {
			v.cheap.Notify(e.Path)
		}
		if e.Expensive {
			v.expensive.Notify(e.Path)
		}
	}
}

func (v *inventoryView) watchChanges(ctx context.Context, agents bool) {
	go v.fetchPeriodically(ctx)
	inventory.WatchConfigured(ctx, v.cache.Paths, agents, v.event, func(err error) { log.Printf("inventory events: %v", err) })
}

func (v *inventoryView) fetchPeriodically(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		for _, path := range v.cache.Paths() {
			v.fetchIfDue(ctx, path)
		}
		timer := time.NewTimer(15 * time.Minute)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// No scanMu: full collection must never block this path.
func (v *inventoryView) refreshChanged(ctx context.Context, path string) bool {
	if ctx.Err() != nil || !slices.Contains(v.cache.Paths(), path) {
		return false
	}
	cache, ok := v.cache.(interface{ UpdateLive(string) (bool, error) })
	if !ok {
		return true
	}
	updated, err := cache.UpdateLive(path)
	if err != nil {
		log.Printf("inventory live refresh %s: %v", path, err)
		return false
	}
	if updated && ctx.Err() == nil {
		snapshot := v.cache.Snapshot()
		table, err := cli.BuildTable(snapshot.Overview)
		v.publishTable(snapshot, table, err)
	}
	return updated
}
