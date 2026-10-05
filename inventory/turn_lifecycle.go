package inventory

import "time"

// Native busy and explicit open turns survive quiet tool/subagent waits. The
// cheap path rereads status, but only trusts the identity verified by a full scan.
func claudeActivity(s Object, st transcriptState, path string, now float64, complete bool) {
	value, source, at, confidence, reason := "unknown", path, st.latest, "inferred", "No verified current Claude turn"
	process := str(obj(s["process"])["value"])
	d, err := readObject(str(s["claude_registry_path"]))
	valid := err == nil && str(d["sessionId"]) == str(s["id"]) && number(d["pid"]) == number(s["claude_registry_pid"]) && str(d["procStart"]) == str(s["claude_registry_start"])
	status := ""
	if valid {
		status = str(d["status"])
		s["native_status"] = status
	}
	// Tasks belong to this process lifetime; old transcript launches must not
	// revive tasks after Claude restarts.
	background := Object{}
	started, _ := time.Parse("Mon Jan _2 15:04:05 2006", str(d["procStart"]))
	for id, launched := range st.background {
		if started.IsZero() || timestamp(launched) >= float64(started.Unix()) {
			background[id] = launched
		}
	}
	s["claude_background_tasks"] = background
	terminal := st.terminal
	if process == "stopped" {
		value, at, reason = "stopped", now, "Registered process ended"
	} else if process == "alive" && valid {
		source, at = str(s["claude_registry_path"]), d["statusUpdatedAt"]
		confidence, reason = "observed", "Native status with verified process identity"
		switch {
		case terminal && len(background) == 0 && timestamp(st.end) >= timestamp(at):
			value, source, at = "idle", path, st.end
		case status == "waiting":
			value = "waiting" // Approval/input waits must not look like background work.
		case len(background) > 0:
			value, terminal = "running", false
			confidence, reason = "inferred", "Unfinished Claude background tasks with verified parent process"
		case status == "busy":
			value, terminal = "running", false
		case st.started && !terminal:
			value, confidence, reason = "running", "inferred", "Open Claude turn with verified process, including tool/subagent waits"
		case status == "idle":
			value = "idle"
		}
	} else if terminal {
		value, reason = "idle", "Transcript turn ended; current process unknown"
	}
	s["activity"] = finding(value, source, at, confidence, reason)
	s["last_turn_status"] = nil
	if terminal {
		s["last_turn_status"] = "completed"
	}
	finishQuestion(s, st.pending, path, st.latest, value == "running" || value == "waiting", terminal, complete)
	if value == "waiting" && len(st.pending) == 0 {
		s["question"] = finding("unknown", source, at, "unknown", "Native waiting status does not distinguish question from approval")
	}
}

// A silent outstanding tool (including wait_agent/exec) keeps the parent turn
// open. Old persisted starts without such evidence remain unknown.
func codexActiveTurn(st transcriptState, now float64) bool {
	return st.lifecycle == "inProgress" && (len(st.tools) > 0 || fresh(first(st.latest, st.end), now))
}
