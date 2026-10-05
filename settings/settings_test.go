package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFoldersPersistAndDiscoveryStaysHidden(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zoo", "talkingduck-settings.json")
	s := &Store{Path: path}
	d, err := s.Load()
	if err != nil || len(d.Folders) != 0 {
		t.Fatal(d, err)
	}
	folder := t.TempDir()
	d, err = s.Discover([]string{folder, folder})
	if err != nil || len(d.Folders) != 1 || d.Folders[0].Visible {
		t.Fatal(d, err)
	}
	if _, err = s.SetVisible(folder, true); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SetShortcut(folder, "19"); err != nil {
		t.Fatal(err)
	}
	other := &Store{Path: path}
	d, err = other.Load()
	if err != nil || len(d.Paths()) != 1 || d.Folders[0].MenubarItemShortcut != "19" {
		t.Fatal(d, err)
	}
	if _, err = other.SetVisible(folder, false); err != nil {
		t.Fatal(err)
	}
	d, err = s.Discover([]string{folder})
	if err != nil || len(d.Paths()) != 0 || d.Folders[0].MenubarItemShortcut != "19" {
		t.Fatal(d, err)
	}
	for _, value := range []string{"", "A", "a", "0", "9", "10", "19"} {
		if !ValidShortcut(value) {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"20", "01", "AA", "!", "á"} {
		if _, err = s.SetShortcut(folder, value); err == nil {
			t.Fatal(value)
		}
	}
	before, _ := os.ReadFile(path)
	if _, err = s.SetVisible(filepath.Join(folder, "missing"), true); err == nil {
		t.Fatal("missing folder accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failure changed settings")
	}
	if err = os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SetShortcut(folder, "A"); err == nil {
		t.Fatal("corrupt settings overwritten")
	}
}
func TestDefaultPathUsesXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if DefaultPath() != filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "zoo", "talkingduck-settings.json") {
		t.Fatal(DefaultPath())
	}
}

func TestConcurrentStoresPreserveEdits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	a, b := &Store{Path: path}, &Store{Path: path}
	one, two := t.TempDir(), t.TempDir()
	done := make(chan error, 2)
	go func() { _, err := a.SetVisible(one, true); done <- err }()
	go func() { _, err := b.SetVisible(two, true); done <- err }()
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	data, err := a.Load()
	if err != nil || len(data.Paths()) != 2 {
		t.Fatal(data, err)
	}
}
