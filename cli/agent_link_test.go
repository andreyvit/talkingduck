package cli

import "testing"

func TestAgentLinks(t *testing.T) {
	const id = "11111111-2222-3333-4444-555555555555"
	for _, tc := range []struct{ engine, client, id, app, want string }{
		{"codex", "desktop", id, "", "codex://threads/" + id},
		{"claude", "desktop", id, "local_" + id, "claude://code/continue?session=local_" + id},
		{"claude", "desktop", id, "", ""},
		{"claude", "cli", id, "local_" + id, ""},
		{"codex", "desktop", "new", "", ""},
		{"grok", "desktop", id, "", ""},
	} {
		var task taskView
		task.PrimarySession.Engine = tc.engine
		task.PrimarySession.Client = tc.client
		task.PrimarySession.ID = tc.id
		task.PrimarySession.AppSessionID = tc.app
		if got := agentLink(task); got != tc.want {
			t.Fatalf("%+v: %q", tc, got)
		}
	}
}
