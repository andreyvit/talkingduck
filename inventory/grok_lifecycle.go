package inventory

// Lifecycle evidence survives output silence. Updates and lifecycle events have
// independent cursors/timestamps; a newer completion still closes an older turn.
func grokLifecycle(s Object, events []Object, st *transcriptState) {
	phase, at, outcome := str(s["grok_phase"]), s["grok_phase_at"], s["grok_outcome"]
	for _, r := range events {
		if timestamp(r["ts"]) < timestamp(at) {
			continue
		}
		switch str(r["type"]) {
		case "turn_started":
			phase, at, outcome = "running", r["ts"], nil
			st.started = true
		case "phase_changed":
			phase, at, outcome = str(r["phase"]), r["ts"], nil
		case "turn_ended":
			phase, at, outcome = "ended", r["ts"], r["outcome"]
		}
	}
	s["grok_phase"], s["grok_phase_at"], s["grok_outcome"] = phase, at, outcome
	if phase == "ended" && timestamp(at) >= timestamp(st.latest) {
		st.terminal, st.end = true, at
		clear(st.pending)
	} else if phase != "" && phase != "ended" && timestamp(at) > timestamp(st.end) {
		st.terminal = false
	}
	if timestamp(at) > timestamp(st.latest) {
		st.latest = at
	}
}
func grokActiveTurn(s Object, st transcriptState) bool {
	phase := str(s["grok_phase"])
	return str(obj(s["process"])["value"]) == "alive" && phase != "" && phase != "ended" && !st.terminal && timestamp(s["grok_phase_at"]) > timestamp(st.end)
}
