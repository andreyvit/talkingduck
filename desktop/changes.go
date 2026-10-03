//go:build darwin && cgo

package main

import (
	"context"
	"log"
	"strings"
	"time"

	"talkingduck/cli"
	"talkingduck/inventory"
)

// Polling hints coalesce per folder. A scan covers pending hints only after
// its start; signals arriving during a scan remain eligible for a later refresh.
func (v *inventoryView) watchChanges(ctx context.Context, agents bool) {
	probe := inventory.NewChangeProbe(v.cache.Paths(), "", agents)
	pending := map[string]time.Time{}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	lastWarning := ""
	for {
		if ctx.Err() != nil {
			return
		}
		changed, warnings := probe.Poll()
		now := time.Now()
		for _, path := range changed {
			pending[path] = now
		}
		if warning := strings.Join(warnings, "; "); warning != lastWarning {
			if warning != "" {
				log.Printf("inventory change signals: %s", warning)
			}
			lastWarning = warning
		}
		snapshot := v.cache.Snapshot()
		for _, repo := range snapshot.Details.Repositories {
			signaled, ok := pending[repo.Path]
			if !ok {
				continue
			}
			covered := repo.Cache != nil && (!repo.Cache.LiveObservedAt.Before(signaled))
			if covered {
				delete(pending, repo.Path)
				continue
			}
			if ctx.Err() != nil {
				return
			}
			if v.refreshChanged(ctx, repo.Path) {
				delete(pending, repo.Path)
			}

		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// No scanMu: full collection must never block this path.
func (v *inventoryView) refreshChanged(ctx context.Context, path string) bool {
	if ctx.Err() != nil {
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
