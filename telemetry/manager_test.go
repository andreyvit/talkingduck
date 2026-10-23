package telemetry

import (
	"context"
	"testing"
	"time"
)

func TestSuspendCancelsAndResumesPing(t *testing.T) {
	started := make(chan string, 10)
	canceled := make(chan struct{}, 10)
	probe := func(ctx context.Context, target string) (time.Duration, error) {
		started <- target
		<-ctx.Done()
		canceled <- struct{}{}
		return 0, ctx.Err()
	}
	config := Config{PingEnabled: true, PingTarget: "192.0.2.1", Codex: "off", Claude: "off", Grok: "off"}
	m := newManager(config, nil, probe)
	defer m.Close()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("not started")
	}
	m.SetSuspended(true)
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("not canceled")
	}
	config.PingTarget = "192.0.2.2"
	m.Update(config)
	select {
	case <-started:
		t.Fatal("started while suspended")
	case <-time.After(1100 * time.Millisecond):
	}
	m.SetSuspended(false)
	select {
	case target := <-started:
		if target != config.PingTarget {
			t.Fatal(target)
		}
	case <-time.After(time.Second):
		t.Fatal("not resumed")
	}
	m.SetSuspended(false)
	select {
	case <-canceled:
		t.Fatal("duplicate resume restarted collector")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestSuspendedStartupDoesNotCollect(t *testing.T) {
	started := make(chan struct{}, 1)
	m := createManager(Config{PingEnabled: true, Codex: "off", Claude: "off", Grok: "off"}, nil, func(context.Context, string) (time.Duration, error) { started <- struct{}{}; return 0, nil }, true)
	defer m.Close()
	select {
	case <-started:
		t.Fatal("collector started before display awake")
	case <-time.After(20 * time.Millisecond):
	}
	m.SetSuspended(false)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("collector not resumed")
	}
}
