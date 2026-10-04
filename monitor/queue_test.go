package monitor

import (
	"context"
	"testing"
	"time"
)

func TestDue(t *testing.T) {
	base := time.Unix(1000, 0)
	cases := []struct {
		name   string
		p      pending
		policy Policy
		want   time.Duration
	}{
		{"quiet", pending{first: base, last: base}, Cheap, time.Second},
		{"continuous cap", pending{first: base, last: base.Add(4900 * time.Millisecond)}, Cheap, 5 * time.Second},
		{"finish throttle", pending{first: base, last: base, finished: base.Add(3 * time.Second)}, Cheap, 8 * time.Second},
		{"expensive quiet", pending{first: base, last: base}, Expensive, 5 * time.Second},
		{"expensive cap", pending{first: base, last: base.Add(19 * time.Second)}, Expensive, 20 * time.Second},
		{"expensive throttle", pending{first: base, last: base, finished: base}, Expensive, 60 * time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.p.due(c.policy).Sub(base); got != c.want {
				t.Fatalf("%s != %s", got, c.want)
			}
		})
	}
}
func TestEventsDuringProcessingAndIndependentQueues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	expensive := NewQueue(Policy{time.Millisecond, 3 * time.Millisecond, 40 * time.Millisecond})
	cheap := NewQueue(Policy{})
	started := make(chan string, 8)
	release := make(chan struct{})
	finished := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		expensive.Run(ctx, func(_ context.Context, k string) error {
			started <- k
			if k == "a" {
				select {
				case <-release:
				case <-ctx.Done():
				}
			}
			return nil
		})
	}()
	cheapDone := make(chan struct{})
	go func() {
		defer close(cheapDone)
		cheap.Run(ctx, func(context.Context, string) error { close(finished); return nil })
	}()
	expensive.Notify("a")
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("no start")
	}
	expensive.Notify("a")
	expensive.Notify("a")
	expensive.Notify("b")
	cheap.Notify("a")
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("cheap blocked")
	}
	close(release)
	select {
	case k := <-started:
		if k != "b" {
			t.Fatalf("throttle ignored: %s", k)
		}
	case <-time.After(time.Second):
		t.Fatal("other folder blocked")
	}
	select {
	case <-started:
		t.Fatal("throttle measured from start")
	case <-time.After(15 * time.Millisecond):
	}
	select {
	case k := <-started:
		if k != "a" {
			t.Fatal(k)
		}
	case <-time.After(time.Second):
		t.Fatal("event during processing lost")
	}
	select {
	case <-started:
		t.Fatal("duplicates not coalesced")
	case <-time.After(50 * time.Millisecond):
	}
	cancel()
	<-done
	<-cheapDone
}
func TestRequestCancellation(t *testing.T) {
	q := NewQueue(Expensive)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		q.Run(ctx, func(context.Context, string) error { t.Error("unexpected processing"); return nil })
	}()
	result := make(chan error, 1)
	go func() { result <- q.Request(context.Background(), "a") }()
	cancel()
	<-done
	select {
	case err := <-result:
		if err != context.Canceled {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("request stranded")
	}
}
