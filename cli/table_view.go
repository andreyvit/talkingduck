package cli

import (
	"bytes"
	"strings"
)

// TableView is the CLI's unwrapped table, also used by the desktop overview.
// Each row is one task (or empty/error checkout); newlines stay within cells.
type TableView struct {
	Rows     [][]string `json:"rows"`
	Warnings []string   `json:"warnings"`
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
	if text := strings.TrimSpace(warnings.String()); text != "" {
		result.Warnings = strings.Split(text, "\n")
	}
	return result, nil
}
