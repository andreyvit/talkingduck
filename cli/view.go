package cli

import "encoding/json"

// Only fields used by terminal output belong here. Collection owns the full
// schema and evidence; this projection never infers or changes inventory state.
type tableReport struct {
	Repositories []worktreeView `json:"repositories"`
	AgentSources []sourceView   `json:"agent_sources"`
	AgentError   string         `json:"agent_error"`
}
type sourceView struct {
	Engine   string   `json:"engine"`
	Status   string   `json:"status"`
	Warnings []string `json:"warnings"`
}
type worktreeView struct {
	UnassignedCommits []string   `json:"unassigned_commits"`
	Path              string     `json:"path"`
	Git               gitView    `json:"git"`
	Specs             []taskView `json:"specs"`
	Warnings          []string   `json:"warnings"`
}
type gitView struct {
	Tracked       any      `json:"tracked"`
	Untracked     any      `json:"untracked"`
	Informational any      `json:"informational"`
	Ahead         any      `json:"ahead"`
	Behind        any      `json:"behind"`
	Cleanliness   string   `json:"cleanliness"`
	Error         string   `json:"error"`
	Warnings      []string `json:"warnings"`
}
type taskView struct {
	Path           string   `json:"path"`
	Tickets        []string `json:"tickets"`
	DisplayTitle   string   `json:"display_title"`
	PrimarySession struct {
		Engine       string `json:"engine"`
		ID           string `json:"id"`
		AppSessionID string `json:"app_session_id"`
		Client       string `json:"client"`
	} `json:"primary_session"`
	PrimaryConflict bool         `json:"primary_conflict"`
	Workflow        workflowView `json:"workflow"`
}
type workflowView struct {
	Round         any    `json:"round"`
	Stage         string `json:"stage"`
	SubtasksDone  any    `json:"subtasks_done"`
	SubtasksTotal any    `json:"subtasks_total"`
	Progress      string `json:"progress"`
	Interaction   string `json:"interaction"`
	Questions     any    `json:"questions"`
	Suggestions   any    `json:"suggestions"`
}

func tableData(data any) (tableReport, error) {
	var result tableReport
	b, err := json.Marshal(data)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(b, &result)
	return result, err
}
