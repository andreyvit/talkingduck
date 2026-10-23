package inventory

import (
	"context"
	"slices"
	"sync"
	"time"
)

// WatchConfigured replaces event sources when membership changes. Queues and
// their cooldown/throttle history remain owned by the caller. Old sources are
// stopped before new ones start; hidden folders are filtered by queue workers.
func WatchConfigured(ctx context.Context, paths func() []string, agents bool, emit func(ChangeEvent), warn func(error)) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	watchConfigured(ctx, paths, agents, emit, warn, ticker.C, nil)
}

// WatchConfiguredChanges uses explicit membership notifications instead of polling.
func WatchConfiguredChanges(ctx context.Context, paths func() []string, agents bool, emit func(ChangeEvent), warn func(error), changed <-chan struct{}) {
	watchConfigured(ctx, paths, agents, emit, warn, nil, changed)
}

func watchConfigured(ctx context.Context, paths func() []string, agents bool, emit func(ChangeEvent), warn func(error), ticks <-chan time.Time, changed <-chan struct{}) {
	var current []string
	var stop context.CancelFunc
	var workers sync.WaitGroup
	restart := func(next []string) {
		if stop != nil {
			stop()
			workers.Wait()
		}
		current = append([]string(nil), next...)
		child, cancel := context.WithCancel(ctx)
		stop = cancel
		workers.Add(1)
		go func() { defer workers.Done(); WatchChanges(child, next, "", agents, 10*time.Minute, emit, warn) }()
		if agents {
			workers.Add(1)
			go func() { defer workers.Done(); NewProcessChanges(next, "").Run(child, emit) }()
		}
	}
	restart(paths())
	defer func() { stop(); workers.Wait() }()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
		case <-changed:
		}
		if ctx.Err() != nil {
			return
		}
		next := paths()
		if !slices.Equal(next, current) {
			restart(next)
		}
	}
}
