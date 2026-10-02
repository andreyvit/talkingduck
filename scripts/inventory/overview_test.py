import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

module = importlib.util.spec_from_file_location("overview", Path(__file__).with_name("overview.py"))
o = importlib.util.module_from_spec(module)
module.loader.exec_module(o)


def spec(path=".spec/20261002-dev-1234-example.md", **extra):
    return dict(path=path, title="20261002-dev-1234-example", tickets=["DEV-1234"], status="building high-level plan", **extra)


def session(**extra):
    return dict(engine="grok", id="one", client="cli", source="fixture", title="DEV-1234 work",
                activity={"value": "idle"}, question={"value": "no"}, updated_at="2026-10-02T00:00:00Z", **extra)


class OverviewTests(unittest.TestCase):
    def test_only_subtasks_count_and_formats(self):
        text = "## Report\n- [x] duplicate\n## Subtasks\n1. [x] One\n    - [ ] Nested\n### [>] 2. Two\n- [ ] 3. Three\n## Log\n- [x] other\n"
        w = o.task_state(spec(), text)
        self.assertEqual((w["subtasks_done"], w["subtasks_total"]), (1, 3))

    def test_chat_skill_invocation_and_completion(self):
        rows = [("user", "Run [$zoo-rebase](/skills/zoo-rebase/SKILL.md)", "2026-10-02T00:00:00Z"),
                ("assistant", "Rebase finished.", "2026-10-02T00:10:00Z")]
        result = o.skill_activity(rows)
        self.assertEqual((result["stage"], result["state"]), ("REBA", "completed"))
        rows.append(("user", "Implement the new revision", None))
        self.assertIsNone(o.skill_activity(rows))
        self.assertIsNone(o.skill_activity([("user", "Documentation mentions zoo-rebase.", None)]))
        self.assertIsNone(o.skill_activity([("user", "Do not run zoo-push", None)]))
        rows = [("user", "$zoo-push", None), ("assistant", "Rebase completed as part of push.", None)]
        self.assertEqual(o.skill_activity(rows)["stage"], "PUSH")

    def test_rebase_context(self):
        task = spec(); task["status"] = "done"
        body = "## Subtasks\n1. [x] Work\n## Log\n"
        for entry, expected in [("Rebase started", "REBA"), ("Rebase finished", "REBA"), ("Rebase completed as part of push", "PUSH"), ("Rebase completed during Zoo closeout", "FIN"), ("Started zoo-push\n- Rebase finished", "PUSH"), ("Zoo closeout started\n- Rebase finished", "FIN")]:
            with self.subTest(entry=entry):
                self.assertEqual(o.task_state(task, body + "- " + entry + "\n")["stage"], expected)

    def test_squash_is_per_revision_and_not_commit_count(self):
        task = spec(); task.update(status="done", commits=["one"])
        body = "## Subtasks\n1. [x] Implement\n## Log\n- Completed subtask 1\n"
        self.assertEqual(o.task_state(task, body)["stage"], "FIN")
        body += "- Squashed all task commits.\n"
        w = o.task_state(task, body)
        self.assertEqual((w["round"], w["stage"], w["squashed_this_round"]), (1, "SQ", True))
        body += "- User revision: change the output format.\n"
        w = o.task_state(task, body)
        self.assertEqual((w["round"], w["stage"], w["squashed_this_round"]), (2, "FIN", False))
        body += "- Squashed the revision commits.\n"
        w = o.task_state(task, body)
        self.assertEqual((w["round"], w["stage"]), (2, "SQ"))
        body += "- Final review started.\n"
        self.assertEqual(o.task_state(task, body)["stage"], "FIN")

    def test_review_rounds_do_not_increment_task_round(self):
        task = spec()
        body = "## Log\n- High-level spec review round 5 passed.\n- Low-level spec review round 8 passed.\n"
        self.assertEqual(o.task_state(task, body)["round"], 1)
        self.assertEqual(o.task_state(task, "## High-level plan\n<placeholder>\n")["round"], 0)

    def test_unfinished_subtasks_prevent_sq(self):
        task = spec(); task["status"] = "executing subtask 2"
        body = "## Subtasks\n1. [x] First\n2. [ ] Second\n## Log\n- Squashed old work.\n"
        w = o.task_state(task, body)
        self.assertEqual(w["stage"], "EX")
        self.assertFalse(w["squashed_this_round"])

    def test_suggestions_ignore_templates_and_resolved(self):
        text = "## Pending suggestions\n### Code changes\n<template>\n- First\n  - detail\n- [x] Resolved\n- [ ] Second\n## Log\n"
        self.assertEqual(o.task_state(spec(), text)["suggestions"], 2)
        self.assertIsNone(o.task_state(spec(), "## Pending suggestions\nSome unstructured concern.\n")["suggestions"])

    def test_low_level_over_high_level_approval(self):
        s = spec(); s["status"] = "high-level plan approved, low-level plan in spec review"
        self.assertEqual(o.task_state(s, "")["stage"], "LL")

    def test_secondary_review_cannot_be_primary(self):
        s = session(); s["client"] = "headless"
        with patch.object(o, "transcript_evidence", return_value=([("user", "Review .spec/20261002-dev-1234-example.md. Findings only; do not modify files.", None)], False)):
            self.assertEqual(o.associate(s, [spec()]), spec()["path"])
        self.assertEqual(s["task_role"]["value"], "secondary")

    def test_ambiguous_paths_do_not_pick_latest_spec(self):
        s = session(); s["title"] = "General work"
        other = spec(".spec/other.md")
        text = spec()["path"] + " and " + other["path"]
        with patch.object(o, "transcript_evidence", return_value=([("assistant", text, None)], False)):
            self.assertIsNone(o.associate(s, [spec(), other]))
        self.assertEqual(s["spec_association"]["value"], "ambiguous")

    def test_unique_title_corroborates_multiple_references(self):
        s = session(); other = spec(".spec/other.md"); other["tickets"] = ["DEV-9999"]
        with patch.object(o, "transcript_evidence", return_value=([("assistant", spec()["path"] + " " + other["path"], None)], False)):
            self.assertEqual(o.associate(s, [spec(), other]), spec()["path"])

    def test_tool_output_and_instructions_do_not_associate(self):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d) / "log.jsonl"
            p.write_text(json.dumps({"type": "response_item", "payload": {"type": "function_call_output", "output": spec()["path"]}}) + "\n")
            s = session(transcript_path=str(p)); s.update(engine="codex", title="General work")
            self.assertIsNone(o.associate(s, [spec()]))

    def test_git_counts_staged_unstaged_and_info(self):
        repo = dict(path="fixture", local_changes=[dict(path=".spec/x.md", status=" M"), dict(path="a.go", status="M "), dict(path="new.go", status="??"), dict(path=".proposals/p.md", status="??")])
        with patch.object(o, "git", return_value="1\t2\n") as query:
            g = o.git_state(repo)
        self.assertEqual((g["tracked"], g["untracked"], g["informational"], g["behind"], g["ahead"]), (2, 1, 2, 1, 2))
        query.assert_called_once_with("fixture", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")

    def test_clean_checkout_never_reads_stashes(self):
        with patch.object(o, "git", return_value="8\t0\n") as query:
            result = o.git_state(dict(path="fixture", local_changes=[]))
        self.assertEqual(result["cleanliness"], "CLEAN")
        self.assertEqual(result["tracked"], 0)
        self.assertNotIn("stashes", result)
        query.assert_called_once_with("fixture", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")

    def test_latest_primary_and_conflict_secondary_excluded(self):
        s = spec()
        old = session(); old["activity"] = {"value": "running"}
        new = session(); new.update(engine="claude", id="two", updated_at="2026-10-03T00:00:00Z"); new["activity"] = {"value": "running"}
        review = session(); review.update(client="headless", id="review", updated_at="2026-10-04T00:00:00Z")
        report = dict(repositories=[dict(path="/fixture", specs=[s], agents=[old, new, review])])
        with patch.object(o, "git_state", return_value={}), patch.object(o, "transcript_evidence", return_value=([("user", "Review " + s["path"] + ". Findings only; do not modify files.", None)], False)):
            o.enrich(report)
        self.assertEqual(s["primary_session"], dict(engine="claude", id="two"))
        self.assertTrue(s["primary_conflict"])
        self.assertEqual(len(s["sessions"]), 3)

    def test_multiline_tickets_top_aligned_and_empty_worktree(self):
        s = spec(); s.update(tickets=["DEV-1234", "DEV-5678"], display_title="example", primary_session=None, primary_conflict=False)
        s["workflow"] = dict(stage="HL", subtasks_done=0, subtasks_total=0, progress="?", interaction="?", questions=None, suggestions=0)
        git = dict(tracked=0, untracked=0, informational=1, ahead=0, behind=0, cleanliness="CLEAN")
        result = o.render(dict(repositories=[dict(path="/back1", git=git, specs=[s]), dict(path="/back2", git=git, specs=[])]))
        lines = result.splitlines()
        self.assertEqual(lines[1].strip(), "DEV-5678")
        self.assertIn("CLEAN", lines[2])
        self.assertLessEqual(max(map(o.cell_width, lines)), 120)


if __name__ == "__main__":
    unittest.main()
