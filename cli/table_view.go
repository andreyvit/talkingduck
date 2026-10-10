package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"talkingduck/table"
)

// TableView is the CLI's unwrapped table, also used by the desktop overview.
// Each row is one task (or empty/error checkout); newlines stay within cells.
type TableView struct {
	AgentChoices  [][]table.Item    `json:"-"`
	Parents       map[string]string `json:"-"`
	Model         table.Table       `json:"table"`
	WorktreePaths []string          `json:"worktreePaths"` // Absolute worktree path for every row, even when its label is blank.
	Rows          [][]string        `json:"rows"`
	SpecPaths     []string          `json:"specPaths"` // One absolute path per row; empty for non-task rows.
	AgentLinks    []string          `json:"agentLinks"`
	Warnings      []string          `json:"warnings"`
}

// BuildTable uses the same projection and row builder as terminal output.
func BuildTable(overview any) (TableView, error) {
	report, err := tableData(overview)
	if err != nil {
		return TableView{}, err
	}
	var warnings bytes.Buffer
	printWarnings(&warnings, report)
	result := TableView{Rows: tableRows(report), Warnings: []string{}, Parents: map[string]string{}}
	for _, repo := range report.Repositories {
		result.Parents[repo.Path] = repo.ParentPath
		if repo.HideSummary && (repo.Git.Error != "" || len(repo.Specs) == 0) {
			continue
		}
		count := len(repo.Specs)
		if needsCurrentSummary(repo) {
			count++
		}
		if repo.Git.Error != "" {
			count = 1
		}
		for range count {
			result.WorktreePaths = append(result.WorktreePaths, repo.Path)
		}
		if repo.Git.Error != "" || needsCurrentSummary(repo) {
			result.SpecPaths = append(result.SpecPaths, "")
			result.AgentLinks = append(result.AgentLinks, "")
			result.AgentChoices = append(result.AgentChoices, nil)
			if repo.Git.Error != "" {
				continue
			}
		}
		for _, spec := range repo.Specs {
			path := spec.Path
			if path != "" && !filepath.IsAbs(path) {
				path = filepath.Join(repo.Path, path)
			}
			result.SpecPaths = append(result.SpecPaths, path)
			result.AgentLinks = append(result.AgentLinks, agentLink(spec))
			result.AgentChoices = append(result.AgentChoices, agentChoices(spec))
		}
	}
	if text := strings.TrimSpace(warnings.String()); text != "" {
		result.Warnings = strings.Split(text, "\n")
	}
	result.Model = modelTable(result)
	return result, nil
}
