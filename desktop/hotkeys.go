package main

import (
	"errors"
	"fmt"
	"sync"
	"talkingduck/settings"
)

type shortcutRegistry interface {
	Register(string, func()) error
	Unregister(string) error
}

// Serializes recording, registration, and persistence. New bindings are acquired
// before old bindings are released; rejected edits leave the last setting intact.
type hotkeyController struct {
	mu             sync.Mutex
	registry       shortcutRegistry
	store          *settings.Store
	action         func()
	markdownAction func()
	keys           settings.Hotkeys
	active         map[string]bool
	recording      bool
}

func (h *hotkeyController) replace(keys settings.Hotkeys) error {
	if err := settings.ValidateHotkeys(keys); err != nil {
		return err
	}
	added := []string{}
	for _, key := range keys {
		if key == "" || h.active[key] {
			continue
		}
		if err := h.registry.Register(key, func() {
			h.mu.Lock()
			action := h.action
			if key == h.keys[2] {
				action = h.markdownAction
			}
			h.mu.Unlock()
			if action != nil {
				action()
			}
		}); err != nil {
			for _, key := range added {
				_ = h.registry.Unregister(key)
				delete(h.active, key)
			}
			return fmt.Errorf("%s: %w", key, err)
		}
		if h.active == nil {
			h.active = map[string]bool{}
		}
		h.active[key] = true
		added = append(added, key)
	}
	for key := range h.active {
		if key == keys[0] || key == keys[1] || key == keys[2] {
			continue
		}
		if err := h.registry.Unregister(key); err != nil {
			return err
		}
		delete(h.active, key)
	}
	h.keys = keys
	return nil
}
func (h *hotkeyController) handle(kind string, keys settings.Hotkeys) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	data, err := h.store.Load()
	if err != nil {
		return err
	}
	previous := data.EffectiveHotkeys()
	switch kind {
	case "record":
		if err := h.replace(settings.Hotkeys{}); err != nil {
			return err
		}
		h.recording = true
		return nil
	case "save":
		h.recording = false
		if err := h.replace(keys); err != nil {
			restore := h.replace(previous)
			if restore != nil {
				return fmt.Errorf("%v; restore: %w", err, restore)
			}
			return err
		}
		if _, err := h.store.SetHotkeys(keys); err != nil {
			return errors.Join(err, h.replace(previous))
		}
		return nil
	case "sync":
		if h.recording {
			return nil
		}
	case "cancel":
		h.recording = false
	default:
		return fmt.Errorf("unknown hotkey action")
	}
	return h.replace(previous)
}
