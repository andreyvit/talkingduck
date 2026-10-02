package cli

import "regexp"

var sessionUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var claudeDesktopID = regexp.MustCompile(`^local_[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func agentLink(task taskView) string {
	p := task.PrimarySession
	switch p.Engine {
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
