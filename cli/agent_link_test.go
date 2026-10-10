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
		{"grok", "desktop", id, "", "talkingduck-grok:" + id},
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

func TestAgentChoicesLatestPrimary(t *testing.T) {
	const older = "11111111-2222-3333-4444-555555555555"
	const newer = "22222222-2222-3333-4444-555555555555"
	var task taskView
	for _, data := range []struct{ engine, id, role, at string }{
		{"codex", older, "primary", "2026-10-01T00:00:00Z"},
		{"codex", newer, "primary", "2026-10-02T00:00:00Z"},
		{"codex", older, "secondary", "2026-10-03T00:00:00Z"},
		{"claude", older, "primary", "2026-10-01T00:00:00Z"},
		{"grok", newer, "primary", "2026-10-01T00:00:00Z"},
	} {
		s := sessionView{Engine: data.engine, ID: data.id, Client: "desktop", AppSessionID: "local_" + data.id, UpdatedAt: data.at}
		s.TaskRole.Value = data.role
		task.Sessions = append(task.Sessions, s)
	}
	got := agentChoices(task)
	if len(got) != 3 || got[0].Chord != "Ax" || got[0].Link.Target != "codex://threads/"+newer ||
		got[1].Chord != "Ac" || got[2].Chord != "Ag" {
		t.Fatalf("%+v", got)
	}
	task.Sessions = nil
	if len(agentChoices(task)) != 0 {
		t.Fatal("invented session")
	}
}
