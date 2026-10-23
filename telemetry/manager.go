// Package telemetry collects connectivity and account quotas independently of work inventory.
package telemetry

import (
	"context"
	"sync"
	"time"
)

type Config struct {
	PingTarget          string
	PingEnabled         bool
	Codex, Claude, Grok string
}
type Ping struct {
	Digits      int        `json:"digits"`
	Target      string     `json:"target"`
	State       string     `json:"state"`
	RTTMS       *float64   `json:"rtt_ms,omitempty"`
	FailedSince *time.Time `json:"failed_since,omitempty"`
	ObservedAt  time.Time  `json:"observed_at"`
}
type Provider struct {
	ID               string     `json:"id"`
	State            string     `json:"state"`
	RecentlyUsed     bool       `json:"recently_used"`
	LoggedIn         bool       `json:"logged_in"`
	RemainingPercent *float64   `json:"remaining_percent,omitempty"`
	ResetsAt         *time.Time `json:"resets_at,omitempty"`
	ObservedAt       time.Time  `json:"observed_at"`
	Error            string     `json:"error,omitempty"`
	identity         string
	retryAfter       time.Duration
}
type Snapshot struct {
	Ping      Ping       `json:"ping"`
	Providers []Provider `json:"providers"`
}
type Manager struct {
	probe      func(context.Context, string) (time.Duration, error)
	mu         sync.Mutex
	config     Config
	snapshot   Snapshot
	changed    func(Snapshot)
	cancel     context.CancelFunc
	generation uint64
	closed     bool
	suspended  bool
	wg         sync.WaitGroup
	updates    chan struct{}
	done       chan struct{}
	notifyWG   sync.WaitGroup
}

func New(config Config, changed func(Snapshot)) *Manager {
	return newManager(config, changed, ping)
}

// NewSuspended waits for the display lifecycle before starting any collectors.
func NewSuspended(config Config, changed func(Snapshot)) *Manager {
	return createManager(config, changed, ping, true)
}
func newManager(config Config, changed func(Snapshot), probe func(context.Context, string) (time.Duration, error)) *Manager {
	return createManager(config, changed, probe, false)
}
func createManager(config Config, changed func(Snapshot), probe func(context.Context, string) (time.Duration, error), suspended bool) *Manager {
	m := &Manager{suspended: suspended, probe: probe, changed: changed, updates: make(chan struct{}, 1), done: make(chan struct{})}
	m.notifyWG.Add(1)
	go m.dispatch()
	m.Update(config)
	return m
}
func (m *Manager) Snapshot() Snapshot { m.mu.Lock(); defer m.mu.Unlock(); return clone(m.snapshot) }
func clone(s Snapshot) Snapshot {
	s.Providers = append([]Provider(nil), s.Providers...)
	if s.Ping.RTTMS != nil {
		v := *s.Ping.RTTMS
		s.Ping.RTTMS = &v
	}
	if s.Ping.FailedSince != nil {
		v := *s.Ping.FailedSince
		s.Ping.FailedSince = &v
	}
	for i := range s.Providers {
		p := &s.Providers[i]
		if p.RemainingPercent != nil {
			v := *p.RemainingPercent
			p.RemainingPercent = &v
		}
		if p.ResetsAt != nil {
			v := *p.ResetsAt
			p.ResetsAt = &v
		}
	}
	return s
}

// Callbacks run serially, outside the state lock, and must not call Close.
func (m *Manager) dispatch() {
	defer m.notifyWG.Done()
	for {
		select {
		case <-m.done:
			return
		case <-m.updates:
			if m.changed != nil {
				m.changed(m.Snapshot())
			}
		}
	}
}
func (m *Manager) notify() {
	select {
	case m.updates <- struct{}{}:
	default:
	}
}

func (m *Manager) SetSuspended(suspended bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.suspended == suspended {
		return
	}
	m.suspended = suspended
	if suspended {
		if m.cancel != nil {
			m.cancel()
			m.cancel = nil
		}
		m.generation++
	} else {
		m.updateLocked(m.config)
	}
}

func (m *Manager) Update(c Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateLocked(c)
}
func (m *Manager) updateLocked(c Config) {
	if m.closed || (m.cancel != nil && m.config == c) {
		return
	}
	if m.suspended {
		m.config = c
		return
	}
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.config = c
	m.generation++
	g := m.generation
	if c.PingTarget == "" {
		c.PingTarget = "8.8.8.8"
	}
	m.snapshot = Snapshot{Ping: Ping{Target: c.PingTarget, State: "off"}}
	if c.PingEnabled {
		m.snapshot.Ping.State = "checking"
	}
	modes := []string{c.Codex, c.Claude, c.Grok}
	for i, id := range []string{"codex", "claude", "grok"} {
		state := "loading"
		if modes[i] == "off" {
			state = "off"
		}
		m.snapshot.Providers = append(m.snapshot.Providers, Provider{ID: id, State: state})
	}
	if c.PingEnabled {
		m.wg.Add(1)
		go func() { defer m.wg.Done(); m.runPing(ctx, g, c.PingTarget) }()
	}
	for i, p := range m.snapshot.Providers {
		if modes[i] != "off" {
			m.wg.Add(1)
			go func(i int, id, mode string) { defer m.wg.Done(); m.runProvider(ctx, g, i, id, mode) }(i, p.ID, modes[i])
		}
	}
	m.notify()
}
func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	close(m.done)
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	m.wg.Wait()
	m.notifyWG.Wait()
}
func (m *Manager) publish(g uint64, f func(*Snapshot)) {
	m.mu.Lock()
	if m.closed || g != m.generation {
		m.mu.Unlock()
		return
	}
	f(&m.snapshot)
	m.notify()
	m.mu.Unlock()
}
func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
func (m *Manager) runProvider(ctx context.Context, g uint64, index int, id, mode string) {
	interval := time.Minute
	if id == "claude" {
		interval = 5 * time.Minute
	}
	backoff := interval
	for {
		p := collect(ctx, id, mode)
		if ctx.Err() != nil {
			return
		}
		m.publish(g, func(s *Snapshot) { s.Providers[index] = merge(s.Providers[index], p) })
		delay := interval
		if p.State == "error" {
			delay = backoff
			backoff *= 2
			if backoff > 15*time.Minute {
				backoff = 15 * time.Minute
			}
		} else {
			backoff = interval
		}
		if p.retryAfter > delay {
			delay = p.retryAfter
		}
		if !wait(ctx, delay) {
			return
		}
	}
}
func merge(old, p Provider) Provider {
	if p.State == "error" && p.identity != "" && old.identity == p.identity && old.RemainingPercent != nil {
		p.State = "stale"
		p.RemainingPercent = old.RemainingPercent
		p.ResetsAt = old.ResetsAt
		p.ObservedAt = old.ObservedAt
	}
	return p
}
func (m *Manager) runPing(parent context.Context, g uint64, target string) {
	ctx, cancel := context.WithCancel(parent)
	var probes sync.WaitGroup
	defer func() { cancel(); probes.Wait() }()
	type result struct {
		sequence uint64
		started  time.Time
		rtt      time.Duration
		err      error
	}
	results := make(chan result, 8)
	var sequence, accepted uint64
	var failed *time.Time
	var widths pingWidths
	launch := func(start time.Time) {
		sequence++
		seq := sequence
		probes.Add(1)
		go func() {
			defer probes.Done()
			rtt, err := m.probe(ctx, target)
			select {
			case results <- result{seq, start, rtt, err}:
			case <-ctx.Done():
			}
		}()
	}
	previous := time.Now()
	launch(previous)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			if now.Sub(previous) > 5*time.Second {
				failed = nil
				widths = pingWidths{}
				accepted = sequence
			}
			previous = now
			launch(now)
		case r := <-results:
			// A delayed reply must not replace a newer observation.
			if r.sequence <= accepted {
				continue
			}
			accepted = r.sequence
			p := Ping{Target: target, State: "reachable", ObservedAt: time.Now()}
			if r.err != nil {
				if failed == nil {
					start := r.started
					failed = &start
				}
				p.State = "unreachable"
				p.FailedSince = failed
			} else {
				failed = nil
				ms := float64(r.rtt) / float64(time.Millisecond)
				p.RTTMS = &ms
			}
			p.Digits = widths.observe(p.ObservedAt, p.RTTMS)
			m.publish(g, func(s *Snapshot) { s.Ping = p })
		}
	}
}

// Each threshold expires ten seconds after the last reading at or above it.
type pingWidths struct{ seen [3]time.Time }

func (w *pingWidths) observe(now time.Time, ms *float64) int {
	if ms != nil {
		for i, threshold := range []float64{10, 100, 1000} {
			if *ms >= threshold {
				w.seen[i] = now
			}
		}
	}
	for i := 2; i >= 0; i-- {
		if !w.seen[i].IsZero() && now.Sub(w.seen[i]) < 10*time.Second {
			return i + 2
		}
	}
	return 1
}
