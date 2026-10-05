package inventory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const liveTailBytes = 256 * 1024

type logCursor struct {
	info   os.FileInfo
	offset int64
	stamp  string
	anchor []byte
}

// Read only newly appended complete lines. Rotation/truncation or in-place
// replacement reseeds from a bounded tail; an incomplete last line stays pending.
func (c *logCursor) read(path string) ([]Object, bool, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, false, e
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil {
		return nil, false, e
	}
	stamp := fileSignal(path)
	if c.info != nil && c.stamp == stamp {
		return nil, false, nil
	}
	reset := c.info == nil || !os.SameFile(c.info, st) || st.Size() < c.offset || (st.Size() == c.offset && stamp != c.stamp) || st.Size()-c.offset > liveTailBytes
	if !reset && len(c.anchor) > 0 {
		anchor := make([]byte, len(c.anchor))
		if _, err := f.ReadAt(anchor, c.offset-int64(len(anchor))); err != nil || !bytes.Equal(anchor, c.anchor) {
			reset = true
		}
	}
	start := c.offset
	if reset {
		start = max(int64(0), st.Size()-liveTailBytes)
	}
	if _, e = f.Seek(start, io.SeekStart); e != nil {
		return nil, false, e
	}
	b, e := io.ReadAll(io.LimitReader(f, liveTailBytes))
	if e != nil {
		return nil, false, e
	}
	skip := 0
	if reset && start > 0 {
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			skip = i + 1
		} else {
			skip = len(b)
		}
	}
	end := bytes.LastIndexByte(b, '\n') + 1
	if end < skip {
		end = skip
	}
	rows, _ := parseRows(b[skip:end], false)
	c.info, c.offset, c.stamp = st, start+int64(end), stamp
	c.anchor = make([]byte, min(int64(64), c.offset))
	if len(c.anchor) > 0 {
		_, _ = f.ReadAt(c.anchor, c.offset-int64(len(c.anchor)))
	}
	return rows, reset, nil
}

type liveSession struct {
	cursor logCursor
	events logCursor
	chat   logCursor
	state  transcriptState
}
type liveState struct{ sessions map[string]*liveSession }

// UpdateLive refreshes known specs/session evidence with cached Git and ownership.
// It never runs subprocesses or rescans history. False means the
// entry has no baseline yet; caller can retry on the next poll.
func (c *Inventory) UpdateLive(path string) (bool, error) {
	path, e := filepath.Abs(path)
	if e != nil {
		return false, e
	}
	entry := c.entries[path]
	if entry == nil {
		return false, fmt.Errorf("worktree is not configured: %s", path)
	}
	entry.liveMu.Lock()
	defer entry.liveMu.Unlock()
	c.mu.RLock()
	base := entry.snapshot
	c.mu.RUnlock()
	if base == nil {
		return false, nil
	}
	b, e := json.Marshal(base)
	if e != nil {
		return false, e
	}
	var next Snapshot
	if e = json.Unmarshal(b, &next); e != nil {
		return false, e
	}
	if entry.live == nil {
		entry.live = &liveState{sessions: map[string]*liveSession{}}
	}
	now := time.Now().UTC()
	refreshLiveSnapshot(&next, entry.live, now)
	c.mu.Lock()
	c.generation++
	entry.snapshot = &next
	entry.refresh.LiveObservedAt = now
	c.mu.Unlock()
	return true, nil
}
func (l *liveState) update(s Object, now time.Time) {
	path := str(s["transcript_path"])
	engine := str(s["engine"])
	if engine == "grok" {
		path = filepath.Join(filepath.Dir(path), "updates.jsonl")
	}
	if path == "" {
		return
	}
	key := engine + ":" + str(s["id"])
	state := l.sessions[key]
	if state == nil {
		state = &liveSession{}
		l.sessions[key] = state
	}
	rows, reset, e := state.cursor.read(path)
	if e != nil {
		return
	}
	if reset || state.state.pending == nil {
		firstRead := state.state.pending == nil
		state.state = transcriptState{pending: map[string]Object{}}
		if engine == "claude" && firstRead {
			// The full scan can see launches older than the cheap bounded tail.
			state.state.background = Object{}
			for id, at := range obj(s["claude_background_tasks"]) {
				state.state.background[id] = at
			}
		}
	}
	if len(rows) == 0 && engine != "grok" && engine != "claude" {
		return
	}
	switch engine {
	case "codex":
		state.state = codexTranscript(s, rows, state.state)
		if state.state.lifecycle == "" {
			state.state.lifecycle = str(s["last_turn_status"]) // Bounded tail may omit the turn start.
		}
	case "claude":
		state.state = claudeTranscript(s, rows, state.state)
	case "grok":
		state.state = grokUpdates(s, rows, state.state)
	default:
		return
	}
	// Grok publishes lifecycle events independently of update chunks.
	if engine == "grok" {
		events, resetEvents, eventErr := state.events.read(filepath.Join(filepath.Dir(path), "events.jsonl"))
		if eventErr == nil {
			if resetEvents {
				delete(s, "grok_phase")
				delete(s, "grok_phase_at")
				delete(s, "grok_outcome")
			}
			grokLifecycle(s, events, &state.state)
		}
		chat, _, _ := state.chat.read(str(s["transcript_path"]))
		appendLiveReceipts(s, chat)
	} else {
		appendLiveReceipts(s, rows)
	}
	st := state.state
	if engine == "claude" {
		claudeActivity(s, st, path, float64(now.UnixNano())/1e9, !reset || st.started)
		s["updated_at"] = iso(st.latest)
		return
	}
	terminal := st.terminal || contains([]string{"completed", "failed", "interrupted"}, st.lifecycle)
	at := first(st.latest, st.end)
	live := fresh(at, float64(now.UnixNano())/1e9)
	value := "unknown"
	if engine == "grok" && str(obj(s["process"])["value"]) == "stopped" {
		value = "stopped"
	} else if terminal {
		value = "idle"
	} else if (engine == "grok" && grokActiveTurn(s, st)) || (engine == "codex" && codexActiveTurn(st, float64(now.UnixNano())/1e9)) || (live && engine != "codex") {
		value = "running"
	}
	s["activity"] = finding(value, path, at, "inferred", "Incremental transcript evidence; process liveness retained from full scan")
	s["updated_at"] = iso(at)
	if terminal {
		s["last_turn_status"] = "completed"
	} else if engine == "grok" {
		s["last_turn_status"] = nil
	}
	finishQuestion(s, st.pending, path, at, (value == "running") && !terminal, terminal, !reset || st.started)
}

func refreshLiveSnapshot(next *Snapshot, live *liveState, now time.Time) {
	if len(objects(next.Overview["repositories"])) == 0 {
		return
	}
	repo := objects(next.Overview["repositories"])[0]
	for i, s := range objects(repo["specs"]) {
		specPath := str(s["path"])
		text, e := os.ReadFile(filepath.Join(str(repo["path"]), specPath))
		if e != nil {
			continue
		} // Full discovery reconciles deletions; keep prior evidence meanwhile.
		parsed := parseSpec(specPath, string(text))
		s["title"], s["display_title"], s["status"], s["tickets"] = parsed.Title, rx(`(?i)^\d{8}-(?:(?:dev-)?\d+-)*`).ReplaceAllString(parsed.Title, ""), parsed.Status, parsed.Tickets
		s["workflow"] = taskState(s, string(text))
		if i < len(next.Details.Repositories[0].Specs) {
			old := next.Details.Repositories[0].Specs[i]
			old.Title, old.Status, old.Tickets, old.MentionedTickets = parsed.Title, parsed.Status, parsed.Tickets, parsed.MentionedTickets
		}
		for _, session := range objects(s["sessions"]) {
			live.update(session, now)
		}
		finishTask(s)
	}
	repo["live_observed_at"] = now
}

func appendLiveReceipts(s Object, rows []Object) {
	entries := []conversationEntry{}
	for _, row := range rows {
		var text string
		if str(s["engine"]) == "codex" {
			p := obj(row["payload"])
			if str(row["type"]) == "event_msg" && str(p["type"]) == "agent_message" {
				text = textContent(p["message"])
			}
			if str(row["type"]) == "response_item" && str(p["type"]) == "message" && str(p["role"]) == "assistant" {
				text = textContent(p["content"])
			}
		} else if str(row["type"]) == "assistant" {
			if m := obj(row["message"]); len(m) > 0 {
				text = textContent(m["content"])
			} else {
				text = textContent(row["content"])
			}
		}
		if text != "" {
			entries = append(entries, conversationEntry{role: "assistant", text: text, at: row["timestamp"]})
		}
	}
	receipts := objects(s["zoo_receipts"])
	for _, r := range zooReceipts(entries) {
		duplicate := false
		for _, old := range receipts {
			if old["evidence"] == r["evidence"] && old["at"] == r["at"] {
				duplicate = true
				break
			}
		}
		if !duplicate {
			receipts = append(receipts, r)
		}
	}
	s["zoo_receipts"] = receipts
}
