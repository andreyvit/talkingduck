package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type transcriptState struct {
	pending           map[string]Object
	lifecycle         string
	latest, end       any
	started, terminal bool
}

func fresh(at any, now float64) bool { t := timestamp(at); return t != 0 && now-t >= -5 && now-t < 120 }
func codexTranscript(s Object, rows []Object) transcriptState {
	st := transcriptState{pending: map[string]Object{}}
	for _, r := range rows {
		p := obj(r["payload"])
		at := r["timestamp"]
		typ := str(p["type"])
		switch str(r["type"]) {
		case "event_msg":
			switch typ {
			case "task_started", "turn_started":
				st.lifecycle, st.end, st.started = "inProgress", at, true
				clear(st.pending)
			case "task_complete", "turn_complete", "turn_aborted":
				st.lifecycle = "completed"
				if typ == "turn_aborted" {
					st.lifecycle = "interrupted"
				}
				st.end = at
				clear(st.pending)
			case "user_message":
				clear(st.pending)
				message(s, "user", at, p["message"])
			case "agent_message":
				message(s, "assistant", at, p["message"])
			}
		case "response_item":
			switch typ {
			case "function_call", "custom_tool_call":
				q := questionCall(str(p["name"]), str(p["call_id"]), first(p["arguments"], p["input"]), at)
				if q != nil {
					st.pending[str(q["call_id"])] = q
				}
			case "function_call_output", "custom_tool_call_output":
				id := str(p["call_id"])
				if truth(st.pending[id]["blocking"]) {
					delete(st.pending, id)
				}
			case "message":
				role := str(p["role"])
				if role == "user" || role == "assistant" {
					message(s, role, at, p["content"])
				}
			}
		}
		if timestamp(at) != 0 {
			st.latest = at
		}
	}
	return st
}
func claudeTranscript(s Object, rows []Object) transcriptState {
	st := transcriptState{pending: map[string]Object{}}
	for _, r := range rows {
		typ, at := str(r["type"]), r["timestamp"]
		if truth(at) {
			st.latest = at
		}
		m := obj(r["message"])
		content := m["content"]
		if typ == "user" || typ == "assistant" {
			message(s, typ, at, content)
		}
		for _, v := range array(content) {
			b := obj(v)
			switch str(b["type"]) {
			case "tool_use":
				st.terminal = false
				q := questionCall(str(b["name"]), str(b["id"]), b["input"], at)
				if q != nil {
					st.pending[str(q["call_id"])] = q
				}
			case "tool_result":
				delete(st.pending, str(b["tool_use_id"]))
			}
		}
		if typ == "assistant" && str(m["stop_reason"]) == "end_turn" {
			st.terminal = true
		} else if typ == "user" && strings.TrimSpace(cleanText(content)) != "" {
			st.terminal = false
			clear(st.pending)
		}
	}
	return st
}
func grokUpdates(s Object, rows []Object) transcriptState {
	st := transcriptState{pending: map[string]Object{}}
	streamRole, streamText := "", ""
	for _, r := range rows {
		p := obj(r["params"])
		u := obj(p["update"])
		at := first(obj(p["_meta"])["agentTimestampMs"], r["timestamp"])
		if truth(at) {
			st.latest = at
		}
		kind := str(u["sessionUpdate"])
		switch kind {
		case "user_message_chunk", "agent_message_chunk":
			role := "assistant"
			if kind == "user_message_chunk" {
				role = "user"
			}
			if role != streamRole {
				streamRole, streamText = role, ""
			}
			streamText = runeSuffix(streamText+cleanText([]Object{obj(u["content"])}), 16000)
			if role == "user" {
				st.terminal = false
				clear(st.pending)
			}
			message(s, role, at, streamText)
		case "tool_call":
			streamRole, streamText = "", ""
			q := questionCall(str(u["title"]), str(u["toolCallId"]), u["rawInput"], at)
			if q != nil {
				st.pending[str(q["call_id"])] = q
			}
		case "tool_call_update":
			if contains([]string{"completed", "failed", "cancelled"}, str(u["status"])) {
				delete(st.pending, str(u["toolCallId"]))
			}
		case "turn_completed":
			st.terminal, st.end = true, at
			clear(st.pending)
		}
	}
	return st
}

type processProbe func(Object, string) Object

func probeProcess(record Object, engine string) Object {
	pid := int(number(record["pid"]))
	if pid <= 0 {
		return unknown("No process ID")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ps", "-p", fmt.Sprint(pid), "-o", "lstart=,comm=")
	cmd.Env = append(os.Environ(), "TZ=UTC", "LC_ALL=C")
	b, e := cmd.Output()
	now := float64(time.Now().UnixNano()) / 1e9
	source := fmt.Sprintf("ps:%d", pid)
	if e != nil {
		if x, ok := e.(*exec.ExitError); ok && x.ExitCode() == 1 && len(strings.TrimSpace(string(b))) == 0 {
			return finding("stopped", source, now, "observed", "Registered process no longer exists")
		}
		return unknown("Process check unavailable")
	}
	m := rx(`^(.{24})\s+(.+)`).FindStringSubmatch(strings.TrimSpace(string(b)))
	if len(m) != 3 {
		return unknown("Unrecognized ps format")
	}
	start, executable := m[1], m[2]
	if engine == "claude" {
		if str(record["procStart"]) == "" || strings.Join(strings.Fields(start), " ") != strings.Join(strings.Fields(str(record["procStart"])), " ") || !strings.Contains(strings.ToLower(executable), "claude") {
			return unknown("Claude process identity/start time did not match registry")
		}
	} else {
		began, err := time.Parse("Mon Jan _2 15:04:05 2006", start)
		opened := timestamp(record["opened_at"])
		if err != nil || opened == 0 || float64(began.Unix()) > opened+2 || !strings.Contains(strings.ToLower(filepath.Base(executable)), "grok") {
			return unknown("Grok process identity/start time did not match registry")
		}
	}
	return finding("alive", source, now, "observed", "Executable and registered process lifetime match")
}

func collectCodex(home string, roots map[string]bool, limit int, now float64) ([]Object, []string, int) {
	records := map[string]Object{}
	warnings := []string{}
	dbs := append(glob(filepath.Join(home, "state_*.sqlite")), glob(filepath.Join(home, "sqlite", "state_*.sqlite"))...)
	for _, path := range dbs {
		db, e := openReadDB(path)
		if e != nil {
			warnings = append(warnings, "Codex metadata unavailable at "+path+": "+e.Error())
			continue
		}
		cols, e := queryObjects(db, "PRAGMA table_info(threads)")
		present := map[string]bool{}
		for _, c := range cols {
			present[str(c["name"])] = true
		}
		if e != nil || !present["id"] || !present["cwd"] || !present["rollout_path"] {
			warnings = append(warnings, "Unsupported threads schema: "+path)
			db.Close()
			continue
		}
		wanted := []string{}
		for _, k := range []string{"id", "cwd", "title", "name", "updated_at", "updated_at_ms", "archived", "source", "originator", "rollout_path", "agent_path"} {
			if present[k] {
				wanted = append(wanted, k)
			}
		}
		rows, e := queryObjects(db, "SELECT "+strings.Join(wanted, ",")+" FROM threads")
		db.Close()
		if e != nil {
			warnings = append(warnings, "Codex metadata unavailable at "+path+": "+e.Error())
			continue
		}
		for _, d := range rows {
			if strings.Contains(str(d["source"]), "subagent") || (str(d["agent_path"]) != "" && str(d["agent_path"]) != "/root") || !scoped(d["cwd"], roots) {
				continue
			}
			age := timestamp(first(d["updated_at_ms"], d["updated_at"]))
			id := str(d["id"])
			if prev := records[id]; prev == nil || age > number(prev["_sort"]) {
				d["_sort"], d["_db"] = age, path
				records[id] = d
			}
		}
	}
	if len(records) == 0 && len(dbs) == 0 {
		files := []string{}
		_ = filepath.WalkDir(filepath.Join(home, "sessions"), func(p string, d os.DirEntry, e error) error {
			if e == nil && !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(p, ".jsonl") {
				files = append(files, p)
			}
			return nil
		})
		sort.Slice(files, func(i, j int) bool { return modTime(files[i]) > modTime(files[j]) })
		if len(files) > 2000 {
			files = files[:2000]
			warnings = append(warnings, "Codex fallback discovery limited to 2000 recent rollouts")
		}
		for _, p := range files {
			d := obj(firstRow(p)["payload"])
			if !scoped(d["cwd"], roots) || str(d["source"]) == "" || str(d["source"]) == "cli" {
				continue
			}
			d["title"], d["rollout_path"], d["_sort"], d["_db"] = d["id"], p, modTime(p), p
			records[str(d["id"])] = d
		}
	}
	candidates := []Object{}
	for _, d := range records {
		path := str(d["rollout_path"])
		origin := str(first(d["originator"], obj(firstRow(path)["payload"])["originator"]))
		client := "unknown"
		if strings.Contains(strings.ToLower(origin), "desktop") {
			client = "desktop"
		}
		if contains([]string{"cli", "exec"}, str(d["source"])) && client != "desktop" {
			continue
		}
		s := newSession("codex", str(d["id"]), str(d["cwd"]), str(first(d["name"], d["title"])), str(d["_db"]), d["_sort"], client)
		s["transcript_path"], s["_db"] = path, d["_db"]
		candidates = append(candidates, s)
	}
	selected := selectRecent(candidates, limit, nil)
	for _, s := range selected {
		path := str(s["transcript_path"])
		st := transcriptState{pending: map[string]Object{}}
		rows, truncated, bad, e := tailRows(path, tailBytes)
		if e == nil {
			tailWarnings(s, path, truncated, bad)
			st = codexTranscript(s, rows)
		} else {
			truncated = true
			warn(s, "Rollout is unavailable")
		}
		history := filepath.Join(filepath.Dir(str(s["_db"])), "thread_history_1.sqlite")
		delete(s, "_db")
		if !exists(history) {
			history = filepath.Join(home, "thread_history_1.sqlite")
		}
		if exists(history) {
			db, e := openReadDB(history)
			if e == nil {
				turns, err := queryObjects(db, "SELECT status,started_at,completed_at FROM thread_turns WHERE thread_id=? ORDER BY rollout_ordinal DESC LIMIT 1", s["id"])
				db.Close()
				if err != nil {
					warn(s, "Thread history schema unavailable; using rollout")
				} else if len(turns) > 0 {
					t := turns[0]
					at := first(t["completed_at"], t["started_at"])
					if timestamp(st.end) == 0 || timestamp(at) > timestamp(st.end) {
						st.lifecycle, st.end = str(t["status"]), at
					}
				}
			} else {
				warn(s, "Thread history schema unavailable; using rollout")
			}
		}
		s["last_turn_status"] = nullable(st.lifecycle)
		latest := first(st.latest, st.end)
		terminal := contains([]string{"completed", "failed", "interrupted"}, st.lifecycle)
		recent := fresh(latest, now)
		if terminal {
			s["activity"] = finding("idle", path, st.end, "inferred", "Last persisted turn ended; does not prove the thread is currently loaded")
			clear(st.pending)
		} else if st.lifecycle == "inProgress" && recent {
			s["activity"] = finding("running", path, latest, "inferred", "Open turn with recent events; live Desktop ownership not verified")
		} else {
			s["activity"] = finding("unknown", path, latest, "unknown", "Persisted open turn or incomplete history; no recent live evidence")
		}
		finishQuestion(s, st.pending, path, latest, recent && st.lifecycle == "inProgress", terminal, !truncated || st.started)
		s["process"] = unknown("Codex app-server shares many threads; process existence cannot identify a live turn")
	}
	return selected, warnings, len(candidates)
}
func modTime(p string) float64 {
	if s, e := os.Stat(p); e == nil {
		return float64(s.ModTime().UnixNano()) / 1e9
	}
	return 0
}

type registryEntry struct {
	data Object
	path string
}

func collectClaude(home, app string, roots map[string]bool, limit int, now float64, probe processProbe) ([]Object, []string, int) {
	candidates := []Object{}
	warnings := []string{}
	registry := map[string]registryEntry{}
	for _, p := range glob(filepath.Join(home, "sessions", "*.json")) {
		d, e := readObject(p)
		if e != nil {
			warnings = append(warnings, "Unreadable Claude registry: "+filepath.Base(p))
			continue
		}
		if str(d["entrypoint"]) == "claude-desktop" {
			registry[str(d["sessionId"])] = registryEntry{d, p}
		}
	}
	for _, p := range glob(filepath.Join(app, "claude-code-sessions", "*", "*", "local_*.json")) {
		d, e := readObject(p)
		if e != nil {
			warnings = append(warnings, "Unreadable Claude Desktop metadata: "+filepath.Base(p))
			continue
		}
		if !scoped(d["cwd"], roots) {
			continue
		}
		sid := str(first(d["cliSessionId"], d["sessionId"]))
		s := newSession("claude", sid, str(d["cwd"]), str(d["title"]), p, d["lastActivityAt"], "desktop")
		s["app_session_id"] = d["sessionId"]
		candidates = append(candidates, s)
	}
	known := map[string]bool{}
	for _, s := range candidates {
		known[str(s["id"])] = true
	}
	for sid, r := range registry {
		if !known[sid] && scoped(r.data["cwd"], roots) {
			s := newSession("claude", sid, str(r.data["cwd"]), str(r.data["name"]), r.path, r.data["updatedAt"], "desktop")
			s["app_session_id"] = r.data["hostSessionId"]
			candidates = append(candidates, s)
		}
	}
	probes := map[string]Object{}
	extra := map[string]bool{}
	for sid, r := range registry {
		if scoped(r.data["cwd"], roots) {
			probes[sid] = probe(r.data, "claude")
			extra[sid] = str(probes[sid]["value"]) == "alive"
		}
	}
	selected := selectRecent(candidates, limit, extra)
	for _, s := range selected {
		sid := str(s["id"])
		files := glob(filepath.Join(home, "projects", "*", sid+".jsonl"))
		sort.Slice(files, func(i, j int) bool { return modTime(files[i]) > modTime(files[j]) })
		path := str(s["source"])
		st := transcriptState{pending: map[string]Object{}}
		truncated := true
		s["transcript_path"] = nil
		if len(files) > 0 {
			path = files[0]
			s["transcript_path"] = path
			rows, tr, bad, e := tailRows(path, tailBytes)
			truncated = tr
			if e == nil {
				tailWarnings(s, path, tr, bad)
				st = claudeTranscript(s, rows)
			} else {
				warn(s, "Transcript unavailable")
			}
		} else {
			warn(s, "No linked local Claude Code transcript")
		}
		nativeWait := false
		if r, ok := registry[sid]; ok {
			s["process"] = probes[sid]
			s["native_status"] = r.data["status"]
			switch str(probes[sid]["value"]) {
			case "alive":
				status := map[string]string{"busy": "running", "waiting": "waiting", "idle": "idle"}[str(r.data["status"])]
				if status == "" {
					status = "unknown"
				}
				nativeWait = status == "waiting"
				s["activity"] = finding(status, r.path, r.data["statusUpdatedAt"], "observed", "Native status with PID and process-start verification")
			case "stopped":
				s["activity"] = finding("stopped", r.path, now, "observed", "Registered CLI process ended; persisted state may be stale")
			}
		} else if st.terminal {
			s["activity"] = finding("idle", path, st.latest, "inferred", "Latest transcript ends a turn; current process unknown")
		}
		if st.terminal {
			s["last_turn_status"] = "completed"
		}
		finishQuestion(s, st.pending, path, st.latest, nativeWait, st.terminal, !truncated)
		if nativeWait && len(st.pending) == 0 {
			s["question"] = finding("unknown", path, st.latest, "unknown", "Native waiting status does not distinguish question from approval")
		}
	}
	return selected, warnings, len(candidates)
}

func collectGrok(home string, roots map[string]bool, limit int, now float64, probe processProbe) ([]Object, []string, int) {
	candidates := []Object{}
	warnings := []string{}
	registry := map[string]Object{}
	active := filepath.Join(home, "active_sessions.json")
	if b, e := os.ReadFile(active); e == nil {
		var records []Object
		if json.Unmarshal(b, &records) != nil {
			warnings = append(warnings, "Grok active-session registry unavailable")
		} else {
			for _, d := range records {
				registry[str(d["session_id"])] = d
			}
		}
	} else if !os.IsNotExist(e) {
		warnings = append(warnings, "Grok active-session registry unavailable")
	}
	for _, p := range glob(filepath.Join(home, "sessions", "*", "*", "summary.json")) {
		d, e := readObject(p)
		if e != nil {
			warnings = append(warnings, "Unreadable Grok summary: "+filepath.Base(filepath.Dir(p)))
			continue
		}
		info := obj(d["info"])
		if truth(d["parent_session_id"]) || strings.HasPrefix(str(d["session_kind"]), "subagent") || !scoped(info["cwd"], roots) {
			continue
		}
		client := "cli"
		if str(d["session_kind"]) == "headless" {
			client = "headless"
		}
		s := newSession("grok", str(info["id"]), str(info["cwd"]), str(first(d["generated_title"], d["session_summary"])), p, first(d["last_active_at"], d["updated_at"]), client)
		s["session_kind"] = textOr(d["session_kind"], "unknown")
		candidates = append(candidates, s)
	}
	probes := map[string]Object{}
	extra := map[string]bool{}
	for sid, d := range registry {
		if scoped(d["cwd"], roots) {
			probes[sid] = probe(d, "grok")
			extra[sid] = str(probes[sid]["value"]) == "alive"
		}
	}
	selected := selectRecent(candidates, limit, extra)
	for _, s := range selected {
		folder := filepath.Dir(str(s["source"]))
		path := filepath.Join(folder, "updates.jsonl")
		s["transcript_path"] = filepath.Join(folder, "chat_history.jsonl")
		st := transcriptState{pending: map[string]Object{}}
		rows, truncated, bad, e := tailRows(path, tailBytes)
		if e == nil {
			tailWarnings(s, path, truncated, bad)
			st = grokUpdates(s, rows)
		} else {
			truncated = true
			warn(s, "Grok updates unavailable")
		}
		s["last_message_update_at"] = obj(s["last_message"])["at"]
		chat, _, _, e := tailRows(str(s["transcript_path"]), tailBytes)
		if e == nil {
			var last Object
			for _, r := range chat {
				role := str(r["type"])
				text := cleanText(r["content"])
				if (role == "user" || role == "assistant") && strings.TrimSpace(text) != "" {
					previous := obj(s["last_message"])
					excerpt := str(previous["excerpt"])
					var at any
					if str(previous["role"]) == role && excerpt != "" && (strings.Contains(text, runePrefix(excerpt, 200)) || strings.Contains(text, runeSuffix(excerpt, 160))) {
						at = previous["at"]
					}
					last = Object{"at": at, "role": role, "excerpt": runePrefix(text, 400)}
				}
			}
			if last != nil {
				s["last_message"] = last
			}
		}
		phase := ""
		var phaseAt, outcome any
		events, tr, bad, e := tailRows(filepath.Join(folder, "events.jsonl"), tailBytes)
		if e == nil {
			tailWarnings(s, filepath.Join(folder, "events.jsonl"), tr, bad)
			for _, r := range events {
				switch str(r["type"]) {
				case "turn_started":
					phase, phaseAt, outcome = "running", r["ts"], nil
				case "phase_changed":
					phase, phaseAt = str(r["phase"]), r["ts"]
				case "turn_ended":
					phase, phaseAt, outcome = "ended", r["ts"], r["outcome"]
				}
			}
			if phase == "ended" && (timestamp(st.latest) == 0 || timestamp(phaseAt) >= timestamp(st.latest)) {
				st.terminal, st.end = true, phaseAt
				clear(st.pending)
			}
		}
		if phase != "ended" && timestamp(phaseAt) > timestamp(st.end) {
			st.terminal = false
		}
		if timestamp(phaseAt) > timestamp(st.latest) {
			st.latest = phaseAt
		}
		sid := str(s["id"])
		live := extra[sid]
		if p := probes[sid]; p != nil {
			s["process"] = p
		}
		s["last_turn_status"] = outcome
		if !truth(outcome) && st.terminal {
			s["last_turn_status"] = "completed"
		}
		if st.terminal {
			value, reason := "idle", "Last recorded turn ended; current process unverified"
			if live {
				reason = "Last recorded turn ended; process is alive"
			} else if str(obj(s["process"])["value"]) == "stopped" {
				value = "stopped"
			}
			s["activity"] = finding(value, path, st.end, "inferred", reason)
		} else if live {
			value, reason := "unknown", "Live process but no recent turn evidence"
			if fresh(st.latest, now) {
				value, reason = "running", "Live process with recent updates"
			}
			s["activity"] = finding(value, path, st.latest, "inferred", reason)
		} else if str(obj(s["process"])["value"]) == "stopped" {
			s["activity"] = finding("stopped", active, now, "observed", "Registered process no longer exists")
		}
		if live && phase == "permission_prompt" && timestamp(phaseAt) != 0 && now-timestamp(phaseAt) < 120 {
			s["permission"] = finding("unknown", filepath.Join(folder, "events.jsonl"), phaseAt, "inferred", "Permission phase can auto-resolve; inspect current UI before claiming a prompt")
		}
		finishQuestion(s, st.pending, path, st.latest, live, st.terminal, !truncated)
		s["last_event_at"] = iso(st.latest)
	}
	return selected, warnings, len(candidates)
}

func collectAgentData(home string, paths []string, limit int, now float64, probe processProbe) Object {
	roots := map[string]bool{}
	for _, p := range paths {
		roots[resolved(p)] = true
	}
	report := Object{"schema_version": 1, "observed_at": iso(now), "sessions": []Object{}, "sources": []Object{}, "limitations": []string{"No hooks, network requests, agent invocations, or UI inspection.", "Activity and questions are separate. Unknown is not idle or no.", "Local Codex threads, Claude Desktop Code chats, and Grok CLI only; cloud-only chats and Claude Chat/Cowork are not covered.", fmt.Sprintf("Session limit %d per provider/folder (0 means all), plus registered live Claude/Grok sessions; subagents excluded.", limit)}}
	providers := []struct {
		engine, path string
		run          func(string) ([]Object, []string, int)
	}{{"codex", envPath("CODEX_HOME", filepath.Join(home, ".codex")), func(p string) ([]Object, []string, int) { return collectCodex(p, roots, limit, now) }}, {"claude", envPath("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude")), func(p string) ([]Object, []string, int) {
		return collectClaude(p, filepath.Join(home, "Library", "Application Support", "Claude"), roots, limit, now, probe)
	}}, {"grok", envPath("GROK_HOME", filepath.Join(home, ".grok")), func(p string) ([]Object, []string, int) { return collectGrok(p, roots, limit, now, probe) }}}
	sessions, sources := []Object{}, []Object{}
	for _, p := range providers {
		ss, w, count := p.run(p.path)
		sessions = append(sessions, ss...)
		status := "available"
		if !exists(p.path) {
			status = "missing"
		} else if len(w) > 0 {
			status = "partial"
		}
		sources = append(sources, Object{"engine": p.engine, "path": p.path, "status": status, "discovered": count, "returned": len(ss), "warnings": w})
	}
	for _, s := range sessions {
		fields := []string{}
		for _, k := range []string{"activity", "question", "permission", "process", "spec_association"} {
			if str(obj(s[k])["value"]) == "unknown" {
				fields = append(fields, k)
			}
		}
		if !truth(obj(s["last_message"])["at"]) {
			fields = append(fields, "last_message.at")
		}
		if str(s["client"]) == "unknown" {
			fields = append(fields, "client")
		}
		s["unknown_fields"] = fields
	}
	report["sessions"], report["sources"] = sessions, sources
	return report
}
func collectAgents(paths []string, limit int) AgentReport {
	home, _ := os.UserHomeDir()
	m := collectAgentData(home, paths, limit, float64(time.Now().UnixNano())/1e9, probeProcess)
	b, _ := json.Marshal(m)
	var report AgentReport
	if e := json.Unmarshal(b, &report); e != nil {
		report.Error = e.Error()
	}
	return report
}
