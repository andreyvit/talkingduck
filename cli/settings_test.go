package cli

import (
	"flag"
	"os/exec"
	"path/filepath"
	"talkingduck/settings"
	"testing"
)

func TestConfiguredPathsAndExplicitOverride(t *testing.T) {
	opts := RegisterFlags(flag.NewFlagSet("test", flag.ContinueOnError))
	opts.SettingsPath = filepath.Join(t.TempDir(), "settings.json")
	data, err := opts.CollectionOptions(nil)
	if err != nil || len(data.Paths) != 0 {
		t.Fatal(data, err)
	}
	store := settings.Store{Path: opts.SettingsPath}
	folder, _ := settings.NormalizePath(t.TempDir())
	if out, err := exec.Command("git", "init", "-q", folder).CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	if _, err = store.SetVisible(folder, true); err != nil {
		t.Fatal(err)
	}
	data, err = opts.CollectionOptions(nil)
	if err != nil || len(data.Paths) != 1 || data.Paths[0] != folder {
		t.Fatal(data, err)
	}
	override := t.TempDir()
	data, err = opts.CollectionOptions([]string{override})
	if err != nil || len(data.Paths) != 1 || data.Paths[0] != override {
		t.Fatal(data, err)
	}
	saved, err := store.Load()
	if err != nil || saved.Folders[0].Path != folder {
		t.Fatal(saved, err)
	}
}
