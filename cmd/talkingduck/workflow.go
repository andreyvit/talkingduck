package main

import (
	"strconv"
	"strings"
)

func section(text, heading string) string {
	lines := strings.Split(text, "\n")
	inside := false
	out := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if inside {
				break
			}
			inside = strings.EqualFold(strings.TrimSpace(line[3:]), heading)
			continue
		}
		if inside {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}
func taskState(spec Object, text string) Object {
	subtasks := rx(`(?m)^(?:\d+[.)] |[-*] |### )\[([ xX>])\]`).FindAllStringSubmatch(section(text, "Subtasks"), -1)
	done := 0
	for _, m := range subtasks {
		if strings.EqualFold(m[1], "x") {
			done++
		}
	}
	pending := rx(`(?s)<[^>]*>`).ReplaceAllString(section(text, "Pending suggestions"), "")
	items := rx(`(?m)^(?:[-*] |\d+[.)] )(.*\S)`).FindAllStringSubmatch(pending, -1)
	count := 0
	for _, m := range items {
		if !matches(`(?i)^\[[xX]\]|^~~|^(?:None|No pending)\b`, m[1]) {
			count++
		}
	}
	var suggestions any = count
	if len(items) == 0 {
		for _, line := range strings.Split(pending, "\n") {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "*") && !matches(`(?i)^(?:\d+[.)]|None\.?$|No (?:pending|open|suggestions))`, line) {
				suggestions = nil
				break
			}
		}
	}
	status := strings.ToLower(str(spec["status"]))
	log := rx(`(?s)<[^>]*>`).ReplaceAllString(section(text, "Log"), "")
	events := []string{}
	for _, m := range rx(`(?m)^(?:[-*] |\d+[.)] )(.+)$`).FindAllStringSubmatch(log, -1) {
		events = append(events, m[1])
	}
	revisionEvents := []string{}
	for _, e := range events {
		if matches(`(?i)\buser revision\b|\b(?:user|human) (?:requested|requests|asked for) (?:a |another |major |new )*revision\b|\b(?:major|task) (?:revision|iteration) (?:requested|started)\b`, e) {
			revisionEvents = append(revisionEvents, e)
		}
	}
	revisionEvents = unique(revisionEvents)
	substantive := false
	for _, h := range []string{"High-level plan", "Low-level plan", "Product Spec", "Spec"} {
		s := rx(`(?ms)<[^>]*>|^#+[^\n]*$`).ReplaceAllString(section(text, h), "")
		substantive = substantive || strings.TrimSpace(s) != ""
	}
	round := 0
	if len(subtasks) > 0 || substantive {
		round = 1
	}
	for _, e := range events {
		if !matches(`(?i)^(?:created|create|initialized) (?:task|spec)`, e) {
			round = 1
			break
		}
	}
	round += len(revisionEvents)
	explicitRound := 0
	roundPattern := `(?i)\b(?:task round|task iteration|major iteration)\s*[:#]?\s*(\d+)\b`
	for _, e := range events {
		if m := rx(roundPattern).FindStringSubmatch(e); len(m) > 1 {
			explicitRound, _ = strconv.Atoi(m[1])
		}
	}
	round = max(round, explicitRound)
	allDone := len(subtasks) > 0 && done == len(subtasks)
	completed := contains([]string{"done", "complete", "completed"}, status)
	stage := "--"
	switch {
	case completed || allDone:
		stage = "FIN"
	case strings.Contains(status, "final") && strings.Contains(status, "review"):
		stage = "FIN"
	case matches(`subtask|execut|implement`, status):
		stage = "EX"
	case matches(`low-level|low level`, status):
		stage = "LL"
		if matches(`waiting.*before low.level`, status) {
			stage = "HL"
		}
	case matches(`high-level|high level`, status):
		stage = "HL"
	case status == "planning":
		stage = "HL"
		if len(subtasks) > 0 {
			stage = "LL"
		}
	case strings.Contains(status, "rebas"):
		stage = "REBA"
		if strings.Contains(status, "push") {
			stage = "PUSH"
		} else if strings.Contains(status, "closeout") {
			stage = "FIN"
		}
	}
	var latest Object
	squashed := false
	operation := ""
	for _, e := range events {
		low := strings.ToLower(e)
		if contains(revisionEvents, e) || matches(roundPattern, e) {
			squashed = false
			operation = ""
			next := "FIN"
			if matches(`replan|high.level|planning`, low) {
				next = "HL"
			} else if len(subtasks) > 0 && !allDone {
				next = "EX"
			} else if stage == "HL" || stage == "LL" {
				next = stage
			}
			latest = Object{"stage": next, "entry": e}
			continue
		}
		action := ""
		if matches(`\b(?:zoo[- ]closeout|closeout (?:started|running|complete)|(?:started|running|completed) closeout)\b`, low) {
			operation, action = "FIN", "FIN"
		} else if matches(`\b(?:zoo-push|push (?:workflow|started|running)|(?:started|running) push)\b`, low) {
			operation, action = "PUSH", "PUSH"
		}
		rebase := matches(`\b(?:rebased|rebasing|rebase (?:started|running|finished|completed?|done)|(?:started|running|completed|finished) (?:zoo-)?rebase)\b`, low)
		switch {
		case rebase:
			if matches(`(?:part of|during|for) (?:zoo[- ]|the )?push|zoo-push`, low) {
				operation = "PUSH"
			} else if strings.Contains(low, "closeout") {
				operation = "FIN"
			}
			action = operation
			if action == "" {
				action = "REBA"
			}
		case matches(`\b(?:squashed|squeezed)\b|\b(?:squash|squeeze)(?: ran| complete| done| succeeded)\b|\b(?:ran|completed) (?:zoo-)?(?:squash|squeeze)\b`, low) && !matches(`\b(?:before|pre-squash|pre-squeeze|not|never|will|plan to|should)\b`, low):
			action = "SQ"
		case matches(`\b(?:ready (?:to|for) push|awaiting push|push approved)\b`, low):
			action = "PUSH"
		case matches(`\b(?:final|uber)[ -]review\b`, low):
			if allDone || completed {
				action = "FIN"
			} else {
				action = ""
			}
		case matches(`\b(?:subtask \d+ (?:started|complete|done)|(?:started|completed|executing) subtask \d+)\b`, low):
			action = "EX"
			if allDone {
				action = "FIN"
			}
		case matches(`\blow-level (?:plan|spec review)\b`, low) && !matches(`low.level.*(?:placeholder|not started|deferred)|before low.level`, low):
			action = "LL"
		case matches(`\bhigh-level (?:plan|spec review)\b`, low):
			action = "HL"
		}
		if action != "" && !rebase && action != "PUSH" && action != "FIN" {
			operation = ""
		}
		if action == "SQ" && allDone {
			squashed = true
		}
		if action != "" {
			latest = Object{"stage": action, "entry": e}
		}
	}
	if latest != nil {
		action := str(latest["stage"])
		if action == "SQ" || action == "PUSH" {
			if allDone || action == "PUSH" {
				stage = action
			}
		} else if !(allDone || completed) || action == "FIN" || action == "REBA" {
			stage = action
		}
	}
	ready := completed || matches(`awaiting|waiting for|ready|approved$|investigation complete`, status) || stage == "SQ" || stage == "PUSH"
	progress := "?"
	if ready || (latest != nil && matches(`(?i)complete|finished|passed|ready`, str(latest["entry"]))) {
		progress = "✓"
	}
	return Object{"stage": stage, "round": round, "round_evidence": revisionEvents, "round_confidence": "inferred", "squashed_this_round": squashed, "latest_logged_action": latest, "subtasks_done": done, "subtasks_total": len(subtasks), "suggestions": suggestions, "progress": progress, "source": spec["path"], "reason": "Inferred from chronological spec Log, status, and top-level sections; missing history can undercount rounds"}
}

type conversationEntry struct {
	role, text string
	at         any
}

func skillActivity(rows []conversationEntry) Object {
	var current Object
	for _, row := range rows {
		low := strings.ToLower(row.text)
		invoked := rx(`(?:\$|/|\brun\s+(?:\[\$?)?|\buse\s+(?:\[\$?)?|\binvoke\s+)(zoo-(?:rebase|push))\b`).FindStringSubmatch(low)
		if row.role == "user" && len(invoked) > 1 && len([]rune(row.text)) < 5000 && !matches(`\b(?:do not|don't|never|not yet)\b`, low) {
			name := invoked[1]
			stage := "REBA"
			if name == "zoo-push" {
				stage = "PUSH"
			}
			current = Object{"skill": name, "stage": stage, "state": "requested", "at": row.at, "evidence": runePrefix(row.text, 500), "confidence": "inferred"}
			continue
		}
		if current == nil {
			continue
		}
		if row.role == "user" && !matches(`^\s*(?:continue|proceed|retry|yes|ok|okay|thanks)[.!]?\s*$`, low) {
			current = nil
			continue
		}
		if row.role != "assistant" {
			continue
		}
		if strings.Contains(low, "rebas") && matches(`(?:part of|during|within) (?:the )?(?:zoo[- ]?)?closeout`, low) {
			current["stage"] = "FIN"
		}
		if strings.Contains(low, "rebas") && matches(`(?:part of|during|within) (?:the )?(?:zoo[- ]?)?push`, low) {
			current["stage"] = "PUSH"
		}
		completed := matches(`\b(?:pushed|push (?:complete[ds]?|finished|done))\b`, low)
		if str(current["skill"]) == "zoo-rebase" {
			completed = matches(`(?:^rebased\b|\bi(?: have|'ve)? rebased\b|\brebase (?:is |has )?(?:complete[ds]?|finished|done)|(?:completed|finished) (?:the )?(?:zoo-)?rebase)\b`, low)
		}
		if completed && !matches(`\b(?:will|not|never|before|plan to)\b`, low) {
			current["state"], current["at"], current["evidence"] = "completed", row.at, runePrefix(row.text, 500)
		}
	}
	return current
}
