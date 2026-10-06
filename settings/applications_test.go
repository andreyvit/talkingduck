package settings

import (
	"path/filepath"
	"talkingduck/launchers"
	"testing"
)

func TestApplicationSettingsPreserveFolders(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	folder := t.TempDir()
	if _, e := s.SetVisible(folder, true); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SetApplication("editor", launchers.Selection{ID: "vscode"}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SetTracker(launchers.TrackerSelection{ID: "linear", URL: "https://linear.app/acme"}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SetShortcut(folder, "1"); e != nil {
		t.Fatal(e)
	}
	d, e := s.Load()
	if e != nil || len(d.Folders) != 1 || d.Applications.Editor.ID != "vscode" || d.Applications.Tracker.AccountSlug != "acme" {
		t.Fatalf("%+v %v", d, e)
	}
	if _, e = s.SetTracker(launchers.TrackerSelection{ID: "linear", URL: "https://wrong.example"}); e == nil {
		t.Fatal("bad URL saved")
	}
	d, _ = s.Load()
	if d.Applications.Tracker.AccountSlug != "acme" {
		t.Fatal("invalid edit overwrote working config")
	}
}
