package markdownview

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryNavigationAndPersistence(t *testing.T) {
	var h History
	h.Open("/a.md")
	h.Remember("/a.md", Position{"heading", 23, 400})
	h.Open("/b.md")
	h.Open("/b.md")
	if len(h.Visits) != 2 {
		t.Fatal(h)
	}
	if !h.Move(-1) || h.Current() != "/a.md" || h.Position(h.Current()).Top != 400 || h.Move(-1) {
		t.Fatal(h)
	}
	if !h.Move(1) || h.Move(1) {
		t.Fatal(h)
	}
	h.Move(-1)
	h.Open("/c.md")
	if h.Move(1) || len(h.Visits) != 2 {
		t.Fatal(h)
	}
	path := filepath.Join(t.TempDir(), "history.json")
	if e := h.Save(path); e != nil {
		t.Fatal(e)
	}
	loaded, e := LoadHistory(path)
	if e != nil || loaded.Current() != "/c.md" || loaded.Position("/a.md").Heading != "heading" {
		t.Fatal(loaded, e)
	}
	for i := 0; i < 100; i++ {
		h.Open(fmt.Sprintf("/%d.md", i))
	}
	if len(h.Visits) != RecentLimit || len(h.Recent) != RecentLimit {
		t.Fatal(h)
	}
}
func TestRefreshCooldownThrottleAndContinuousWrites(t *testing.T) {
	start := time.Unix(100, 0)
	var r Refresh
	r.Changed(start)
	if r.Due(start.Add(499*time.Millisecond)) || !r.Due(start.Add(500*time.Millisecond)) {
		t.Fatal(r)
	}
	r.Complete(start.Add(500 * time.Millisecond))
	r.Changed(start.Add(time.Second))
	if r.Due(start.Add(2499*time.Millisecond)) || !r.Due(start.Add(2500*time.Millisecond)) {
		t.Fatal(r)
	}
	r = Refresh{}
	for i := 0; i < 20; i++ {
		r.Changed(start.Add(time.Duration(i) * 100 * time.Millisecond))
	}
	if r.Due(start.Add(1999*time.Millisecond)) || !r.Due(start.Add(2*time.Second)) {
		t.Fatal(r)
	}
	r.Complete(start.Add(2 * time.Second))
	if r.Due(start.Add(time.Hour)) {
		t.Fatal("rendered without change")
	}
}
func TestKeyboardControls(t *testing.T) {
	for key, want := range map[string]string{"Escape": "close", " ": "page-down", "ArrowUp": "heading-up", "ArrowDown": "heading-down", "ArrowLeft": "back", "ArrowRight": "forward", "r": "refresh", "c": "toc", "h": "history"} {
		if got := KeyAction(key, false, false); got != want {
			t.Fatal(key, got)
		}
		if KeyAction(key, false, true) != "" {
			t.Fatal("stole modified key")
		}
	}
	if KeyAction(" ", true, false) != "page-up" || KeyAction("q", false, false) != "" {
		t.Fatal("wrong key mapping")
	}
	if NextHeading(-1, 1, 3) != 0 || NextHeading(2, 1, 3) != 2 || NextHeading(0, -1, 3) != 0 || NextHeading(0, 1, 0) != -1 {
		t.Fatal("heading bounds")
	}
}
