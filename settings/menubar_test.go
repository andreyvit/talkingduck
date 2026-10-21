package settings

import (
	"path/filepath"
	"testing"
)

func TestMenubarDefaultsAndValidation(t *testing.T) {
	m := (Menubar{}).Defaults()
	if m.Ping != "off" || m.PingTarget != "8.8.8.8" || m.Codex != "auto" || m.Projects != "on" {
		t.Fatal(m)
	}
	for _, target := range []string{"8.8.8.8", "::1", "dns.example.org"} {
		m.PingTarget = target
		if err := m.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []string{"-x", "https://example.org", "a b", "example..org", "a;ls"} {
		m.PingTarget = target
		if m.Validate() == nil {
			t.Fatal(target)
		}
	}
}
func TestMenubarSavePreservesOtherSettings(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	d, err := s.SetMenubar(Menubar{Codex: "off", Projects: "off"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Version != 1 || d.Menubar.Claude != "auto" {
		t.Fatal(d)
	}
	_, err = s.SetMenubar(Menubar{Codex: "invalid"})
	if err == nil {
		t.Fatal("accepted invalid setting")
	}
	d, err = s.Load()
	if err != nil || d.Menubar.Codex != "off" {
		t.Fatal(d, err)
	}
}
