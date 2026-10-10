//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
	"talkingduck/table"
	"testing"
	"time"
)

func TestMarkdownStampSurvivesRepeatedAtomicSaves(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.md")
	fixed := time.Unix(100, 0)
	save := func() {
		t.Helper()
		tmp := filepath.Join(dir, "new.md")
		if e := os.WriteFile(tmp, []byte("same length"), 0600); e != nil {
			t.Fatal(e)
		}
		if e := os.Chtimes(tmp, fixed, fixed); e != nil {
			t.Fatal(e)
		}
		if e := os.Rename(tmp, path); e != nil {
			t.Fatal(e)
		}
	}
	save()
	first := fileStamp(path)
	os.ReadFile(path)
	if fileStamp(path) != first {
		t.Fatal("read caused refresh loop")
	}
	save()
	second := fileStamp(path)
	if first == second {
		t.Fatal("missed first atomic save")
	}
	save()
	if second == fileStamp(path) {
		t.Fatal("missed second atomic save")
	}
	os.Remove(path)
	if second == fileStamp(path) {
		t.Fatal("missed deletion")
	}
	save()
	if fileStamp(path) == "" {
		t.Fatal("missed recreation")
	}
}

func TestMarkdownWindowTitle(t *testing.T) {
	model := table.Table{Columns: []table.Column{{ID: "folder"}}}
	for _, folder := range []struct{ path, label string }{{"/work/repo", "Work label"}, {"/work/repo/nested", "Nested label"}} {
		model.Rows = append(model.Rows, table.Row{Group: folder.path, Cells: []table.Cell{{Lines: []table.Line{{Items: []table.Item{{Text: folder.label}}}}}}})
	}
	for _, tt := range []struct{ path, want string }{
		{"/work/repo/.spec/file.md", "file.md — Work label"},
		{"/work/repo/nested/.spec/file.md", "file.md — Nested label"},
		{"/work/repo-other/file.md", "file.md — /work/repo-other"},
		{"/tmp/file.md", "file.md — /tmp"},
	} {
		if got := markdownWindowTitle(tt.path, model); got != tt.want {
			t.Errorf("title(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
