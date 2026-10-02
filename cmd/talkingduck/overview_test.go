package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func taskFixture() Object {
	return Object{"path": ".spec/20261002-dev-1234-example.md", "title": "20261002-dev-1234-example", "tickets": []string{"DEV-1234"}, "status": "building high-level plan"}
}
func conversationFixture(t *testing.T, role, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "conversation.jsonl")
	writeJSON(t, p, Object{"type": role, "content": []Object{{"type": "text", "text": text}}})
	return p
}
func overviewSession() Object {
	s := testSession("grok")
	s["client"], s["title"] = "cli", "DEV-1234 work"
	s["question"] = finding("no", "", nil, "inferred", "")
	s["activity"] = finding("idle", "", nil, "inferred", "")
	return s
}
func TestSubtaskAndSuggestionCounts(t *testing.T) {
	text := "## Report\n- [x] duplicate\n## Subtasks\n1. [x] One\n    - [ ] Nested\n### [>] 2. Two\n- [ ] 3. Three\n## Log\n- [x] other\n"
	w := taskState(taskFixture(), text)
	if number(w["subtasks_done"]) != 1 || number(w["subtasks_total"]) != 3 {
		t.Fatal(w)
	}
	text = "## Pending suggestions\n### Code changes\n<template>\n- First\n  - detail\n- [x] Resolved\n- [ ] Second\n## Log\n"
	if number(taskState(taskFixture(), text)["suggestions"]) != 2 {
		t.Fatal("suggestion count")
	}
	if taskState(taskFixture(), "## Pending suggestions\nSome unstructured concern.\n")["suggestions"] != nil {
		t.Fatal("prose counted")
	}
}
func TestRebaseContext(t *testing.T) {
	s := taskFixture()
	s["status"] = "done"
	body := "## Subtasks\n1. [x] Work\n## Log\n"
	cases := map[string]string{"Rebase started": "REBA", "Rebase finished": "REBA", "Rebase completed as part of push": "PUSH", "Rebase completed during Zoo closeout": "FIN", "Started zoo-push\n- Rebase finished": "PUSH", "Zoo closeout started\n- Rebase finished": "FIN"}
	for entry, want := range cases {
		t.Run(entry, func(t *testing.T) {
			if got := taskState(s, body+"- "+entry+"\n")["stage"]; got != want {
				t.Fatal(got, want)
			}
		})
	}
}
func TestRevisionRoundResetsSquash(t *testing.T) {
	s := taskFixture()
	s["status"], s["commits"] = "done", []string{"one"}
	body := "## Subtasks\n1. [x] Implement\n## Log\n- Completed subtask 1\n"
	if taskState(s, body)["stage"] != "FIN" {
		t.Fatal("commit count inferred squash")
	}
	steps := []struct {
		entry, stage string
		round        int
		squashed     bool
	}{{"Squashed all task commits.", "SQ", 1, true}, {"User revision: change output.", "FIN", 2, false}, {"Squashed the revision commits.", "SQ", 2, true}, {"Final review started.", "FIN", 2, true}}
	for _, step := range steps {
		body += "- " + step.entry + "\n"
		w := taskState(s, body)
		if w["stage"] != step.stage || int(number(w["round"])) != step.round || truth(w["squashed_this_round"]) != step.squashed {
			t.Fatal(step, w)
		}
	}
}
func TestPlanningRoundAndStage(t *testing.T) {
	s := taskFixture()
	body := "## Log\n- High-level spec review round 5 passed.\n- Low-level spec review round 8 passed.\n"
	if number(taskState(s, body)["round"]) != 1 {
		t.Fatal("review loops counted as task rounds")
	}
	if number(taskState(s, "## High-level plan\n<placeholder>\n")["round"]) != 0 {
		t.Fatal("placeholder counted")
	}
	s["status"] = "high-level plan approved, low-level plan in spec review"
	if taskState(s, "")["stage"] != "LL" {
		t.Fatal("approved HL masks LL")
	}
	s["status"] = "executing subtask 2"
	w := taskState(s, "## Subtasks\n1. [x] First\n2. [ ] Second\n## Log\n- Squashed old work.\n")
	if w["stage"] != "EX" || truth(w["squashed_this_round"]) {
		t.Fatal(w)
	}
}
func TestChatSkillActivity(t *testing.T) {
	rows := []conversationEntry{{"user", "Run [$zoo-rebase](/skills/zoo-rebase/SKILL.md)", nil}, {"assistant", "Rebase finished.", nil}}
	s := skillActivity(rows)
	if s["stage"] != "REBA" || s["state"] != "completed" {
		t.Fatal(s)
	}
	rows = append(rows, conversationEntry{"user", "Implement the new revision", nil})
	if skillActivity(rows) != nil {
		t.Fatal("stale rebase")
	}
	for _, text := range []string{"Documentation mentions zoo-rebase.", "Do not run zoo-push"} {
		if skillActivity([]conversationEntry{{"user", text, nil}}) != nil {
			t.Fatal(text)
		}
	}
	s = skillActivity([]conversationEntry{{"user", "$zoo-push", nil}, {"assistant", "Rebase completed as part of push.", nil}})
	if s["stage"] != "PUSH" {
		t.Fatal(s)
	}
}
func TestAssociationRoleAndAmbiguity(t *testing.T) {
	s := taskFixture()
	p := str(s["path"])
	agent := overviewSession()
	agent["client"] = "headless"
	agent["transcript_path"] = conversationFixture(t, "user", "Review "+p+". Findings only; do not modify files.")
	if associate(agent, []Object{s}) != p || obj(agent["task_role"])["value"] != "secondary" {
		t.Fatal(agent)
	}
	other := taskFixture()
	other["path"], other["tickets"] = ".spec/other.md", []string{"DEV-9999"}
	agent = overviewSession()
	agent["title"] = "General work"
	agent["transcript_path"] = conversationFixture(t, "assistant", p+" and .spec/other.md")
	if associate(agent, []Object{s, other}) != "" || obj(agent["spec_association"])["value"] != "ambiguous" {
		t.Fatal(agent)
	}
	agent["title"] = "DEV-1234 work"
	if associate(agent, []Object{s, other}) != p {
		t.Fatal("unique title failed to corroborate")
	}
}
func TestNoAssociationFromToolOutputOrInstructions(t *testing.T) {
	s := taskFixture()
	p := filepath.Join(t.TempDir(), "log.jsonl")
	writeJSON(t, p, event("response_item", Object{"type": "function_call_output", "output": s["path"]}, testNow))
	a := testSession("codex")
	a["title"], a["transcript_path"] = "General work", p
	if associate(a, []Object{s}) != "" {
		t.Fatal("tool output establishes ownership")
	}
	a["engine"] = "grok"
	a["transcript_path"] = conversationFixture(t, "user", "<rules>"+str(s["path"])+"</rules>")
	if associate(a, []Object{s}) != "" {
		t.Fatal("rules establish ownership")
	}
	if specReference("other-"+filepath.Base(str(s["path"])), filepath.Base(str(s["path"]))) {
		t.Fatal("suffix matched as spec")
	}
}
func TestLatestPrimaryAndConflict(t *testing.T) {
	s := taskFixture()
	old, newer, review := overviewSession(), overviewSession(), overviewSession()
	old["id"], old["updated_at"], old["activity"] = "one", iso(testNow), Object{"value": "running"}
	newer["id"], newer["engine"], newer["updated_at"], newer["activity"] = "two", "claude", iso(testNow+86400), Object{"value": "running"}
	review["id"], review["client"], review["updated_at"] = "review", "headless", iso(testNow+86400*2)
	path := conversationFixture(t, "user", "Review "+str(s["path"])+". Findings only; do not modify files.")
	for _, a := range []Object{old, newer, review} {
		a["transcript_path"] = path
	}
	root := repo(t)
	r := Object{"repositories": []Object{{"path": root, "specs": []Object{s}, "agents": []Object{old, newer, review}}}}
	enrich(r)
	if !reflect.DeepEqual(s["primary_session"], Object{"engine": "claude", "id": "two"}) || !truth(s["primary_conflict"]) || len(array(s["sessions"])) != 3 {
		t.Fatal(s)
	}
}
func TestGitCountsAndNoStashRead(t *testing.T) {
	root := repo(t)
	command(t, root, "config", "remote.origin.url", "https://example.invalid/repo")
	command(t, root, "config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	command(t, root, "branch", "--set-upstream-to=origin/main")
	r := Object{"path": root, "local_changes": []Object{{"path": ".spec/x.md", "status": " M"}, {"path": "a.go", "status": "M "}, {"path": "new.go", "status": "??"}, {"path": ".proposals/p.md", "status": "??"}}}
	g := gitState(r)
	if number(g["tracked"]) != 2 || number(g["untracked"]) != 1 || number(g["informational"]) != 2 || g["behind"] != 0 || g["ahead"] != 0 {
		t.Fatal(g)
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + log + "'\nprintf '8\\t0\\n'\n"
	if e := os.WriteFile(filepath.Join(dir, "git"), []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	g = gitState(Object{"path": "fixture", "local_changes": []Object{}})
	b, _ := os.ReadFile(log)
	if g["cleanliness"] != "CLEAN" || number(g["tracked"]) != 0 || strings.Contains(string(b), "stash") || strings.Count(string(b), "\n") != 1 {
		t.Fatal(g, string(b))
	}
}
func TestMultilineTableAndUnicode(t *testing.T) {
	s := taskFixture()
	s["tickets"], s["display_title"], s["primary_session"], s["primary_conflict"] = []string{"DEV-1234", "DEV-5678"}, "example", nil, false
	s["workflow"] = Object{"stage": "HL", "round": 1, "subtasks_done": 0, "subtasks_total": 0, "progress": "?", "interaction": "?", "questions": nil, "suggestions": 0}
	g := Object{"tracked": 0, "untracked": 0, "informational": 1, "ahead": 0, "behind": 0, "cleanliness": "CLEAN"}
	r := Object{"repositories": []Object{{"path": "/back1", "git": g, "specs": []Object{s}}, {"path": "/back2", "git": g, "specs": []Object{}}}}
	lines := strings.Split(render(r, 120), "\n")
	if strings.TrimSpace(lines[1]) != "DEV-5678" || !strings.Contains(lines[2], "CLEAN") {
		t.Fatal(lines)
	}
	for _, l := range lines {
		if cellWidth(l) > 120 {
			t.Fatal(l)
		}
	}
	if cellWidth("ℹ︎界e\u0301") != 4 {
		t.Fatal("unicode width")
	}
	t.Setenv("COLUMNS", "150")
	if tableWidth(0) != 150 || tableWidth(90) != 90 {
		t.Fatal("width selection")
	}
}
func TestMalformedEvidenceIsSkipped(t *testing.T) {
	p := filepath.Join(t.TempDir(), "log.jsonl")
	b, _ := json.Marshal(Object{"type": "user", "content": "valid"})
	if e := os.WriteFile(p, append([]byte("bad\nnull\n"), append(b, '\n')...), 0600); e != nil {
		t.Fatal(e)
	}
	rows, _ := transcriptEvidence(Object{"engine": "grok", "transcript_path": p})
	if len(rows) != 1 || rows[0].text != "valid" {
		t.Fatal(rows)
	}
}
