// Package monitor schedules coalesced work independently of event sources.
package monitor

import (
	"context"
	"sync"
	"time"
)

type Policy struct{ Quiet, MaxWait, Throttle time.Duration }

var Cheap = Policy{time.Second, 5 * time.Second, 5 * time.Second}
var Expensive = Policy{5 * time.Second, 20 * time.Second, 60 * time.Second}

type pending struct {
	first, last, finished time.Time
	waiters               []chan error
}

func (p *pending) due(policy Policy) time.Time {
	due := p.last.Add(policy.Quiet)
	if cap := p.first.Add(policy.MaxWait); cap.Before(due) {
		due = cap
	}
	if throttle := p.finished.Add(policy.Throttle); throttle.After(due) {
		due = throttle
	}
	return due
}

// Queue has one worker. Separate queues run independently. Events received while
// processing start a new batch; finishing never clears that batch.
type Queue struct {
	mu      sync.Mutex
	policy  Policy
	pending map[string]*pending
	wake    chan struct{}
	closed  bool
}

func NewQueue(policy Policy) *Queue {
	return &Queue{policy: policy, pending: map[string]*pending{}, wake: make(chan struct{}, 1)}
}
func (q *Queue) add(key string, waiter chan error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		if waiter != nil {
			waiter <- context.Canceled
		}
		return
	}
	p := q.pending[key]
	if p == nil {
		p = &pending{}
		q.pending[key] = p
	}
	now := time.Now()
	if p.first.IsZero() {
		p.first = now
	}
	p.last = now
	if waiter != nil {
		p.waiters = append(p.waiters, waiter)
	}
	select {
	case q.wake <- struct{}{}:
	default:
	}
}
func (q *Queue) Notify(key string) { q.add(key, nil) }

// Request joins the next batch and waits for that batch's result. Cancellation
// stops this caller's wait, not shared work requested by other event sources.
func (q *Queue) Request(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan error, 1)
	q.add(key, done)
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run must be called once. Processing errors go to Request callers; the owner
// decides how to publish errors and whether to request another attempt.
func (q *Queue) Run(ctx context.Context, process func(context.Context, string) error) {
	defer func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.closed = true
		for _, p := range q.pending {
			for _, ch := range p.waiters {
				ch <- context.Canceled
			}
			p.waiters = nil
		}
	}()
	for ctx.Err() == nil {
		q.mu.Lock()
		key := ""
		var due time.Time
		for k, p := range q.pending {
			if p.first.IsZero() {
				continue
			}
			d := p.due(q.policy)
			if key == "" || d.Before(due) || (d.Equal(due) && k < key) {
				key, due = k, d
			}
		}
		if key != "" && !time.Now().Before(due) {
			p := q.pending[key]
			waiters := p.waiters
			p.first, p.last, p.waiters = time.Time{}, time.Time{}, nil
			q.mu.Unlock()
			if ctx.Err() != nil {
				for _, ch := range waiters {
					ch <- ctx.Err()
				}
				return
			}
			err := process(ctx, key)
			q.mu.Lock()
			p.finished = time.Now()
			q.mu.Unlock()
			for _, ch := range waiters {
				ch <- err
			}
			continue
		}
		q.mu.Unlock()
		var timer *time.Timer
		var ready <-chan time.Time
		if key != "" {
			timer = time.NewTimer(time.Until(due))
			ready = timer.C
		}
		select {
		case <-ctx.Done():
		case <-q.wake:
		case <-ready:
		}
		if timer != nil {
			timer.Stop()
		}
	}
}
