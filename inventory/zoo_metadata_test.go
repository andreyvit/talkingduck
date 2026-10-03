package inventory

import "testing"

func TestZooFrontmatterOverridesInference(t *testing.T) {
	text := "---\r\nrevision_round: 0\r\nstage: INV\r\nstage_complete: false\r\ntickets: [DEV-12, 'DEV-34']\r\n---\r\n# Inspect test failures\r\n\r\n## User request\r\nOTHER-99 background.\r\n## Log\r\n- User requested a major revision\r\n"
	s := parseSpec(".spec/dev-999-old-title.md", text)
	if len(s.Tickets) != 2 || s.Tickets[0] != "DEV-12" || s.Title != "Inspect test failures" {
		t.Fatal(s)
	}
	w := taskState(Object{"path": s.Path, "status": "done"}, text)
	if number(w["round"]) != 0 || w["stage"] != "INV" || w["progress"] != "?" || w["round_confidence"] != "explicit" {
		t.Fatal(w)
	}
}
func TestZooMetadataValidation(t *testing.T) {
	for _, text := range []string{"---\nrevision_round: -1\nstage: SQ\nstage_complete: yes\n---\n", "# Task\nrevision_round: 0\nstage: INV\nstage_complete: false", "---\nrevision_round: 0\nrevision_round: 2\n---\n"} {
		w := Object{"round": 3, "stage": "HL", "progress": "✓"}
		applyZooMetadata(w, text)
		if w["round"] != 3 || w["stage"] != "HL" || w["progress"] != "✓" {
			t.Fatal(w)
		}
	}
	for _, value := range []string{"[]", "[DEV-1, \"DEV-2\"]", "\n  - DEV-1\n  - DEV-2"} {
		if _, ok := zooTickets("---\ntickets: " + value + "\n---\n"); !ok {
			t.Fatal(value)
		}
	}
	if _, ok := zooTickets("---\ntickets: [bad]\n---\n"); ok {
		t.Fatal("invalid tickets accepted")
	}
	s := parseSpec(".spec/dev-999.md", "---\ntickets: []\n---\n# No ticket")
	if len(s.Tickets) != 0 {
		t.Fatal(s.Tickets)
	}
}
func TestZooReceiptOwnershipAndExamples(t *testing.T) {
	path := ".spec/test.md"
	rows := []conversationEntry{{role: "user", text: "Zoo task start (" + path + ") r0"}, {role: "assistant", text: "```text\nZoo task start (" + path + ") r0\n```\n> Zoo task start (" + path + ") r0\nHistorical: Zoo task start (" + path + ") r0"}}
	if got := zooReceipts(rows); len(got) != 0 {
		t.Fatal(got)
	}
	rows = append(rows, conversationEntry{role: "assistant", text: "Zoo task resume (" + path + ") r0\nZoo review session (.spec/other.md) r0"})
	p, role := receiptOwnership(zooReceipts(rows), []Object{{"path": path}, {"path": ".spec/other.md"}})
	if p != ".spec/other.md" || role != "secondary" {
		t.Fatal(p, role)
	}
}
func TestZooReceiptOperations(t *testing.T) {
	for _, tc := range []struct{ line, stage, progress string }{
		{"Zoo squash finished (.spec/test.md) r0", "SQ", "✓"},
		{"Zoo squash finished (.spec/test.md) r1", "FIN", "✓"},
		{"Zoo squash failed (.spec/test.md) r0: failed", "FIN", "✓"},
		{"Zoo rebase start (.spec/test.md) r0", "REBA", "?"},
		{"Zoo rebase failed (.spec/test.md) r0: conflicts", "REBA", "?"},
		{"Zoo push rebase finished (.spec/test.md) r0", "PUSH", "?"},
		{"Zoo closeout rebase finished (.spec/test.md) r0", "FIN", "✓"},
		{"Zoo format patchset finished (.spec/test.md) r0", "FIN", "✓"},
		{"Zoo squash finished (.spec/test.md) r0\nZoo subtask 2 start (.spec/test.md) r0", "FIN", "✓"},
		{"Zoo squash finished (.spec/other.md) r0", "FIN", "✓"},
	} {
		w := Object{"stage": "FIN", "round": 0, "round_confidence": "explicit", "stage_complete": true, "progress": "✓", "subtasks_done": 2, "subtasks_total": 2}
		applyZooReceipts(w, Object{"zoo_receipts": zooReceipts([]conversationEntry{{role: "assistant", text: tc.line}})}, ".spec/test.md")
		if w["stage"] != tc.stage || w["progress"] != tc.progress {
			t.Fatal(tc.line, w)
		}
	}
}

func TestZooReceiptAssociationOverridesOldTitle(t *testing.T) {
	current, old := taskFixture(), taskFixture()
	old["path"], old["tickets"] = ".spec/old.md", []string{"DEV-9999"}
	for _, role := range []string{"primary", "secondary"} {
		s := overviewSession()
		s["title"] = "DEV-9999 historical work"
		receipt := "Zoo task switch (" + str(current["path"]) + ") r0"
		if role == "secondary" {
			receipt = "Zoo review session (" + str(current["path"]) + ") r0"
		}
		s["transcript_path"] = conversationFixture(t, "assistant", receipt)
		if got := associate(s, []Object{current, old}); got != str(current["path"]) || obj(s["task_role"])["value"] != role {
			t.Fatal(got, s)
		}
	}
}

func TestExplicitTicketsSelectCommittedSpec(t *testing.T) {
	d := repo(t)
	path := ".spec/old-name.md"
	write(t, d, path, "---\nrevision_round: 0\nstage: FIN\nstage_complete: true\ntickets: [DEV-42]\n---\n# Fix export cells\n")
	base := commit(t, d, "Publish spec")
	command(t, d, "update-ref", "refs/remotes/origin/main", base)
	write(t, d, "export.go", "package export\n")
	h := commit(t, d, "Fix export cells DEV-42")
	r := checked(t, d)
	if len(r.Specs) != 1 || r.Specs[0].Path != path || !contains(r.Specs[0].Commits, h) {
		t.Fatal(r)
	}
}
