package cli

import (
	"regexp"
	"talkingduck/table"
	"time"
)

var sessionUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var claudeDesktopID = regexp.MustCompile(`^local_[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func agentLink(task taskView) string {
	return sessionLink(task.PrimarySession)
}

func sessionLink(p sessionView) string {
	switch p.Engine {
	case "grok":
		if sessionUUID.MatchString(p.ID) {
			return "talkingduck-grok:" + p.ID
		}
	case "codex":
		if p.Client != "cli" && sessionUUID.MatchString(p.ID) {
			return "codex://threads/" + p.ID
		}
	case "claude":
		// This is the desktop's sessionId, never an inferred local_ + transcript ID.
		if p.Client == "desktop" && claudeDesktopID.MatchString(p.AppSessionID) {
			return "claude://code/continue?session=" + p.AppSessionID
		}
	}
	return ""
}

// Agent choices use the latest primary per engine, never review sessions.
func agentChoices(task taskView) []table.Item {
	latest := map[string]sessionView{}
	for _, session := range task.Sessions {
		if session.TaskRole.Value != "primary" {
			continue
		}
		old, found := latest[session.Engine]
		at, _ := time.Parse(time.RFC3339Nano, session.UpdatedAt)
		before, _ := time.Parse(time.RFC3339Nano, old.UpdatedAt)
		if !found || at.After(before) || (at.Equal(before) && session.ID > old.ID) {
			latest[session.Engine] = session
		}
	}
	// Older cached reports may only retain the selected primary.
	if _, found := latest[task.PrimarySession.Engine]; !found {
		latest[task.PrimarySession.Engine] = task.PrimarySession
	}
	var items []table.Item
	for _, agent := range []struct{ engine, chord, name string }{
		{"codex", "Ax", "ChatGPT / Codex"}, {"claude", "Ac", "Claude"}, {"grok", "Ag", "Grok Build"},
	} {
		if link := sessionLink(latest[agent.engine]); link != "" {
			items = append(items, table.Item{Text: agent.name, Chord: agent.chord, Link: &table.Action{Kind: "agent", Target: link}})
		}
	}
	return items
}
