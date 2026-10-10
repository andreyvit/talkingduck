package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHotkeyDefaultsAndClearing(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	d, err := s.Load()
	if err != nil || d.EffectiveHotkeys() != (Hotkeys{"Ctrl+T", ""}) {
		t.Fatalf("default: %+v %v", d, err)
	}
	for _, keys := range []Hotkeys{{"Ctrl+Alt+Shift+Super+T", "F18"}, {"", ""}} {
		if _, err := s.SetHotkeys(keys); err != nil {
			t.Fatal(err)
		}
		d, err = s.Load()
		if err != nil || d.EffectiveHotkeys() != keys {
			t.Fatalf("round trip: %+v %v", d, err)
		}
	}
	b, _ := os.ReadFile(s.Path)
	if _, err := s.SetHotkeys(Hotkeys{"Ctrl+T", "Ctrl+T"}); err == nil {
		t.Fatal("duplicate accepted")
	}
	after, _ := os.ReadFile(s.Path)
	if string(b) != string(after) {
		t.Fatal("invalid edit saved")
	}
}
func TestValidateHotkeys(t *testing.T) {
	for _, keys := range []Hotkeys{{"Fn+T", ""}, {"Ctrl+Ctrl+T", ""}, {"Ctrl+Alt+T", "Alt+Ctrl+T"}, {"F21", ""}, {"Ctrl+", ""}, {"MediaPlay", ""}} {
		if ValidateHotkeys(keys) == nil {
			t.Fatalf("accepted %v", keys)
		}
	}
	for _, key := range []string{"Ctrl+Alt+Shift+Super+T", "Alt+=", "Super+Page Up", "F20", "Space", "Shift+;"} {
		if err := ValidateHotkeys(Hotkeys{key, ""}); err != nil {
			t.Fatal(err)
		}
	}
}
