package cli

import (
	"bytes"
	"path/filepath"
	"strings"
)

// TableView is the CLI's unwrapped table, also used by the desktop overview.
// Each row is one task (or empty/error checkout); newlines stay within cells.
type TableView struct {
	Rows      [][]string `json:"rows"`
	SpecPaths []string   `json:"specPaths"` // One absolute path per row; empty for non-task rows.
	Warnings  []string   `json:"warnings"`
}

// BuildTable uses the same projection and row builder as terminal output.
func BuildTable(overview any) (TableView, error) {
	report, err := tableData(overview)
	if err != nil {
		return TableView{}, err
	}
	var warnings bytes.Buffer
	printWarnings(&warnings, report)
	result := TableView{Rows: tableRows(report), Warnings: []string{}}
	for _, repo := range report.Repositories {
		if repo.Git.Error != "" || len(repo.Specs) == 0 {
			result.SpecPaths = append(result.SpecPaths, "")
			continue
		}
		for _, spec := range repo.Specs {
			path := spec.Path
			if path != "" && !filepath.IsAbs(path) {
				path = filepath.Join(repo.Path, path)
			}
			result.SpecPaths = append(result.SpecPaths, path)
		}
	}
	if text := strings.TrimSpace(warnings.String()); text != "" {
		result.Warnings = strings.Split(text, "\n")
	}
	return result, nil
}
