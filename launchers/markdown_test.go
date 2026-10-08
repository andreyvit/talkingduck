package launchers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkdownCatalog(t *testing.T) {
	for _, platform := range []string{"darwin", "linux", "windows"} {
		apps := Available(MarkdownViewers, platform)
		if apps[0].ID != "default" || apps[1].ID != "text-editor" || apps[len(apps)-1].ID != "custom" {
			t.Fatal(apps)
		}
		for i := 3; i < len(apps)-1; i++ {
			if appLess(apps[i], apps[i-1]) {
				t.Fatal("not alphabetical", apps)
			}
		}
		for _, a := range apps[2 : len(apps)-1] {
			cmd, err := Plan(MarkdownViewers, Selection{ID: a.ID}, platform, "/work/spec with spaces.md", func(s string) (string, error) { return s, nil })
			if err != nil || cmd.Args[len(cmd.Args)-1] != "/work/spec with spaces.md" {
				t.Fatal(a, cmd, err)
			}
		}
	}
}
func TestQuickMacDetection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	app := App{ID: "test", Name: "TalkingDuck Synthetic Viewer"}
	path := filepath.Join(home, "Applications", app.Name+".app", "Contents")
	if macAppPath(app) != "" {
		t.Fatal("detected missing app")
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "Info.plist"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if macAppPath(app) != filepath.Dir(path) {
		t.Fatal("missed app")
	}
}
