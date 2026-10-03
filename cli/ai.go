package cli

import (
	"encoding/json"
	"fmt"
	"strings"
)

const aiKey = "Key: worktree agent tickets round stage done/total progress input questions/suggestions ±tracked ?untracked ℹ︎info ⇣behind ⇡ahead title; ▶working ✓ready !input agent!conflict ?unknown -none; HL=high-level LL=low-level EX=execution FIN=finalization SQ=squash PUSH=push PUSHED=recently-pushed REBA=rebase; session=primary ID spec=spec path."

// Reuse table values but omit padding, color, and wrapping. Keep all tickets on
// one line and identify primary sessions directly for callers enriching unknowns.
func renderAI(report tableReport) string {
	lines := []string{}
	for _, repo := range report.Repositories {
		rows := tableRows(tableReport{Repositories: []worktreeView{repo}})
		for index, row := range rows {
			row[0] = repo.Path
			for i, value := range row {
				if i == 2 {
					value = strings.ReplaceAll(value, "\n", ",")
				}
				value = clean(value)
				if i == 3 || i == 5 || i == 8 || (i >= 9 && i <= 13) {
					value = strings.ReplaceAll(value, " ", "")
				}
				if value == "" {
					value = "-"
				}
				row[i] = value
			}
			if repo.Git.Error == "" && len(repo.Specs) > 0 {
				task := repo.Specs[index]
				if task.Path != "" {
					row = append(row, "spec="+clean(task.Path))
				}
				if task.PrimarySession.ID != "" {
					row = append(row, "session="+clean(task.PrimarySession.ID))
				}
			}
			lines = append(lines, strings.Join(row, " "))
		}
	}
	return strings.Join(append(lines, aiKey), "\n")
}

type sessionList struct {
	Sessions []struct {
		Engine    string `json:"engine"`
		ID        string `json:"id"`
		Cwd       string `json:"cwd"`
		Title     string `json:"title"`
		UpdatedAt any    `json:"updated_at"`
		Activity  struct {
			Value string `json:"value"`
		} `json:"activity"`
		Question struct {
			Value string `json:"value"`
		} `json:"question"`
	} `json:"sessions"`
	Sources []sourceView `json:"sources"`
}

func renderSessions(data any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	var report sessionList
	if err = json.Unmarshal(b, &report); err != nil {
		return "", err
	}
	lines := []string{}
	for _, s := range report.Sessions {
		lines = append(lines, fmt.Sprintf("%s %s session=%s activity=%s question=%s updated=%s %s", clean(s.Cwd), clean(s.Engine), clean(s.ID), clean(s.Activity.Value), clean(s.Question.Value), clean(display(s.UpdatedAt)), clean(s.Title)))
	}
	for _, s := range report.Sources {
		if s.Status != "available" || len(s.Warnings) > 0 {
			lines = append(lines, clean(fmt.Sprintf("source %s %s %s", s.Engine, s.Status, strings.Join(s.Warnings, "; "))))
		}
	}
	return strings.Join(append(lines, "Key: worktree agent session=ID activity=observed state question=yes/no/unknown updated=source time title; unknown is not idle or no."), "\n"), nil
}

// FormatAI renders the same compact text as -ai, without collecting inventory.
func FormatAI(overview any) (string, error) {
	report, err := tableData(overview)
	if err != nil {
		return "", err
	}
	return renderAI(report), nil
}
