package inventory

import (
	"strconv"
	"strings"
)

// Only standalone assistant lines are receipts. Ignore quoted examples and code fences.
func zooReceipts(rows []conversationEntry) []Object {
	out := []Object{}
	for _, row := range rows {
		if row.role != "assistant" {
			continue
		}
		fenced := false
		for _, line := range strings.Split(row.text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") {
				fenced = !fenced
				continue
			}
			if fenced {
				continue
			}
			m := rx(`^Zoo (.+?) (start|resume|switch|finished|failed|recorded|session) \(([^)]+)\) r([0-9]+|\?)(?:: (.*))?$`).FindStringSubmatch(line)
			if len(m) == 0 {
				continue
			}
			out = append(out, Object{"operation": m[1], "state": m[2], "spec": m[3], "round": m[4], "detail": m[5], "at": row.at, "evidence": line})
		}
	}
	return out
}
func receiptOwnership(receipts []Object, specs []Object) (string, string) {
	path, role := "", ""
	for _, r := range receipts {
		op, state := str(r["operation"]), str(r["state"])
		if op == "task" && contains([]string{"start", "resume", "switch"}, state) {
			path, role = str(r["spec"]), "primary"
		}
		if op == "review" && state == "session" {
			path, role = str(r["spec"]), "secondary"
		}
		if role != "secondary" && contains([]string{"squash", "rebase", "push", "push rebase", "closeout rebase"}, op) {
			path, role = str(r["spec"]), "primary"
		}
	}
	for _, s := range specs {
		if str(s["path"]) == path {
			return path, role
		}
	}
	return "", role
}
func applyZooReceipts(w, session Object, path string) {
	if str(w["round_confidence"]) != "explicit" {
		for _, r := range objects(session["zoo_receipts"]) {
			if str(r["spec"]) == path {
				if n, err := strconv.Atoi(str(r["round"])); err == nil {
					w["round"], w["round_confidence"] = n, "receipt"
				}
			}
		}
	}
	var latest Object
	for _, r := range objects(session["zoo_receipts"]) {
		if str(r["spec"]) != path || str(r["round"]) == "?" || str(r["round"]) != strconv.Itoa(int(number(w["round"]))) {
			continue
		}
		op, state := str(r["operation"]), str(r["state"])
		stage := ""
		switch op {
		case "squash":
			// SQ is completed squash, never just an invocation or failed attempt.
			if state == "finished" && number(w["subtasks_total"]) > 0 && number(w["subtasks_done"]) == number(w["subtasks_total"]) && w["stage_complete"] != false {
				stage = "SQ"
			} else {
				latest = nil
			}
		case "rebase":
			stage = "REBA"
		case "push", "push rebase":
			stage = "PUSH"
		case "closeout rebase":
			stage = "FIN"
		case "task":
			if state == "finished" {
				latest = nil
			}
		case "final uberreview", "revision":
			latest = nil
		default:
			if strings.HasPrefix(op, "subtask ") || strings.Contains(op, "planning") {
				latest = nil
			}
		}
		if stage != "" {
			latest = Object{"stage": stage, "receipt": r, "state": state}
		}
	}
	if latest == nil {
		return
	}
	w["stage"], w["receipt_evidence"], w["progress"] = latest["stage"], latest["receipt"], "?"
	if str(latest["state"]) == "finished" && str(obj(latest["receipt"])["operation"]) != "push rebase" && (str(latest["stage"]) != "FIN" || w["stage_complete"] == true) {
		w["progress"] = "✓"
	}
	w["squashed_this_round"] = str(latest["stage"]) == "SQ"
}
