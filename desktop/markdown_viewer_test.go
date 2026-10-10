//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
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
	for _, tt := range []struct{ path, home, want string }{
		{"/Users/test/spec.md", "/Users/test", "~/spec.md"},
		{"/Users/test-other/spec.md", "/Users/test", "/Users/test-other/spec.md"},
		{"/tmp/spec.md", "", "/tmp/spec.md"},
	} {
		if got := markdownWindowTitle(tt.path, tt.home); got != tt.want {
			t.Errorf("title(%q, %q) = %q, want %q", tt.path, tt.home, got, tt.want)
		}
	}
}
