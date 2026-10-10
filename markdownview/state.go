package markdownview

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

const RecentLimit = 50

type Position struct {
	Heading string  `json:"heading"`
	Offset  float64 `json:"offset"`
	Top     float64 `json:"top"`
}
type Visit struct {
	Path     string   `json:"path"`
	Position Position `json:"position"`
}

// History uses browser semantics: opening a new file after Back drops Forward.
// Positions survive that branch in a separate bounded recent-files list.
type History struct {
	Visits  []string `json:"visits"`
	Index   int      `json:"index"`
	Recent  []Visit  `json:"recent"`
	HideTOC bool     `json:"hideTOC"`
}

func (h *History) Current() string {
	if h.Index < 0 || h.Index >= len(h.Visits) {
		return ""
	}
	return h.Visits[h.Index]
}
func (h *History) Open(path string) {
	if h.Current() == path {
		return
	}
	if h.Index >= 0 && h.Index+1 < len(h.Visits) {
		h.Visits = h.Visits[:h.Index+1]
	}
	h.Visits = append(h.Visits, path)
	if len(h.Visits) > RecentLimit {
		h.Visits = h.Visits[len(h.Visits)-RecentLimit:]
	}
	h.Index = len(h.Visits) - 1
	h.Remember(path, h.Position(path))
}
func (h *History) Move(delta int) bool {
	next := h.Index + delta
	if next < 0 || next >= len(h.Visits) {
		return false
	}
	h.Index = next
	return true
}
func (h *History) Position(path string) Position {
	for _, v := range h.Recent {
		if v.Path == path {
			return v.Position
		}
	}
	return Position{}
}
func (h *History) Remember(path string, p Position) {
	if path == "" || math.IsNaN(p.Top) || math.IsInf(p.Top, 0) || math.IsNaN(p.Offset) || math.IsInf(p.Offset, 0) {
		return
	}
	p.Top = max(0, p.Top)
	for i, v := range h.Recent {
		if v.Path == path {
			h.Recent = append(h.Recent[:i], h.Recent[i+1:]...)
			break
		}
	}
	h.Recent = append([]Visit{{path, p}}, h.Recent...)
	if len(h.Recent) > RecentLimit {
		h.Recent = h.Recent[:RecentLimit]
	}
}
func LoadHistory(path string) (History, error) {
	var h History
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return h, nil
	}
	if e != nil {
		return h, e
	}
	if e = json.Unmarshal(b, &h); e != nil {
		return History{}, e
	}
	if len(h.Visits) > RecentLimit || len(h.Recent) > RecentLimit || h.Index < 0 || len(h.Visits) > 0 && h.Index >= len(h.Visits) {
		return History{}, fmt.Errorf("invalid Markdown history")
	}
	return h, nil
}
func (h *History) Save(path string) error {
	b, e := json.Marshal(h)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".markdown-history-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), path)
}

// Refresh waits for 500 ms of quiet, caps continuous writes at two seconds,
// and spaces completed automatic renders by at least two seconds.
type Refresh struct{ First, Last, Finished time.Time }

func (r *Refresh) Changed(now time.Time) {
	if r.First.IsZero() {
		r.First = now
	}
	r.Last = now
}
func (r *Refresh) Due(now time.Time) bool {
	if r.First.IsZero() {
		return false
	}
	due := r.Last.Add(500 * time.Millisecond)
	if cap := r.First.Add(2 * time.Second); cap.Before(due) {
		due = cap
	}
	if throttle := r.Finished.Add(2 * time.Second); throttle.After(due) {
		due = throttle
	}
	return !now.Before(due)
}
func (r *Refresh) Complete(now time.Time) {
	r.First = time.Time{}
	r.Last = time.Time{}
	r.Finished = now
}

func KeyAction(key string, shift, modified bool) string {
	if modified {
		return ""
	}
	switch key {
	case "Escape":
		return "close"
	case " ":
		if shift {
			return "page-up"
		}
		return "page-down"
	case "ArrowUp":
		return "heading-up"
	case "ArrowDown":
		return "heading-down"
	case "ArrowLeft":
		return "back"
	case "ArrowRight":
		return "forward"
	case "r", "R":
		return "refresh"
	case "h", "H":
		return "history"
	case "c", "C":
		return "toc"
	}
	return ""
}
func NextHeading(current, delta, count int) int {
	if count == 0 {
		return -1
	}
	return max(0, min(count-1, current+delta))
}
