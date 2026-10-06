package settings

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func gitFolder(t *testing.T) string {
	t.Helper()
	path := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", path).CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	normalized, err := NormalizePath(path)
	if err != nil {
		t.Fatal(err)
	}
	return normalized
}
func TestNonGitFoldersExcluded(t *testing.T) {
	plain := t.TempDir()
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if _, err := s.SetVisible(plain, true); err == nil {
		t.Fatal("accepted non-Git folder")
	}
	repo := gitFolder(t)
	d, err := s.Discover([]string{plain, repo})
	if err != nil || len(d.Folders) != 1 || d.Folders[0].Path != repo {
		t.Fatal(d, err)
	}
	if paths := (Data{Folders: []Folder{{Path: plain, Visible: true}}}).Paths(); len(paths) != 0 {
		t.Fatal("legacy non-Git setting monitored", paths)
	}
}
