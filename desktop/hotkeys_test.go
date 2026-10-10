package main

import (
	"fmt"
	"path/filepath"
	"talkingduck/settings"
	"testing"
)

type fakeShortcuts struct {
	active  map[string]func()
	blocked string
}

func (f *fakeShortcuts) Register(key string, fn func()) error {
	if key == f.blocked || f.active[key] != nil {
		return fmt.Errorf("in use")
	}
	f.active[key] = fn
	return nil
}
func (f *fakeShortcuts) Unregister(key string) error { delete(f.active, key); return nil }
func TestHotkeyRecordingAndFailure(t *testing.T) {
	f := &fakeShortcuts{active: map[string]func(){}}
	store := &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	calls := 0
	h := &hotkeyController{registry: f, store: store, action: func() { calls++ }}
	do := func(kind string, keys settings.Hotkeys) {
		t.Helper()
		if err := h.handle(kind, keys); err != nil {
			t.Fatal(err)
		}
	}
	do("sync", settings.Hotkeys{})
	f.active["Ctrl+T"]()
	if calls != 1 {
		t.Fatal("callback")
	}
	do("record", settings.Hotkeys{})
	do("sync", settings.Hotkeys{})
	if len(f.active) != 0 {
		t.Fatal("shortcut interrupts recording")
	}
	do("cancel", settings.Hotkeys{})
	if f.active["Ctrl+T"] == nil {
		t.Fatal("cancel lost shortcut")
	}
	keys := settings.Hotkeys{"Ctrl+Alt+Shift+Super+T", "F18"}
	do("record", settings.Hotkeys{})
	do("save", keys)
	if len(f.active) != 2 {
		t.Fatal("both shortcuts not registered")
	}
	f.active["F18"]()
	if calls != 2 {
		t.Fatal("additional callback")
	}
	f.blocked = "Ctrl+X"
	do("record", settings.Hotkeys{})
	if h.handle("save", settings.Hotkeys{"Ctrl+X", "F18"}) == nil {
		t.Fatal("conflict not returned")
	}
	data, _ := store.Load()
	if data.EffectiveHotkeys() != keys || len(f.active) != 2 || f.active[keys[0]] == nil {
		t.Fatal("conflict did not restore saved shortcuts")
	}
	do("save", settings.Hotkeys{})
	if len(f.active) != 0 {
		t.Fatal("clearing left registrations")
	}
	do("sync", settings.Hotkeys{})
	if len(f.active) != 0 {
		t.Fatal("cleared default came back")
	}
}

func TestMarkdownHotkeyIndependentAndSwap(t *testing.T) {
	f := &fakeShortcuts{active: map[string]func(){}}
	store := &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	main, preview := 0, 0
	h := &hotkeyController{registry: f, store: store, action: func() { main++ }, markdownAction: func() { preview++ }}
	if e := h.handle("save", settings.Hotkeys{"Ctrl+T", "", "Ctrl+M"}); e != nil {
		t.Fatal(e)
	}
	f.active["Ctrl+T"]()
	f.active["Ctrl+M"]()
	if main != 1 || preview != 1 {
		t.Fatal(main, preview)
	}
	if e := h.handle("save", settings.Hotkeys{"Ctrl+M", "", "Ctrl+T"}); e != nil {
		t.Fatal(e)
	}
	f.active["Ctrl+T"]()
	if preview != 2 || main != 1 {
		t.Fatal("swapped callback did not change")
	}
	if e := h.handle("save", settings.Hotkeys{"Ctrl+M", "", "Ctrl+M"}); e == nil {
		t.Fatal("accepted conflict")
	}
	if e := h.handle("record", settings.Hotkeys{}); e != nil {
		t.Fatal(e)
	}
	if len(f.active) != 0 {
		t.Fatal("preview hotkey active during recording")
	}
	if e := h.handle("cancel", settings.Hotkeys{}); e != nil {
		t.Fatal(e)
	}
	f.active["Ctrl+T"]()
	if preview != 3 {
		t.Fatal("lost preview binding")
	}
}
