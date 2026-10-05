//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFolderTarget(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if e := os.WriteFile(file, nil, 0600); e != nil {
		t.Fatal(e)
	}
	if got, e := folderTarget(root, []string{root}); e != nil || got != root {
		t.Fatal(got, e)
	}
	for _, tc := range []struct {
		path  string
		known []string
	}{{root, nil}, {"relative", []string{"relative"}}, {file, []string{file}}, {filepath.Join(root, "gone"), []string{filepath.Join(root, "gone")}}} {
		if _, e := folderTarget(tc.path, tc.known); e == nil {
			t.Fatalf("accepted %s", tc.path)
		}
	}
}
