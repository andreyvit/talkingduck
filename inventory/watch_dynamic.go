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
		case <-ticker.C:
			next := paths()
			if !slices.Equal(next, current) {
				restart(next)
			}
		}
	}
}
