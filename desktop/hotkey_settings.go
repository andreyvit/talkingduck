//go:build darwin && cgo

package main

import (
	"encoding/json"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"talkingduck/settings"
)

func registerHotkeySettings(app *application.App, window *application.WebviewWindow, store *settings.Store, action func()) func() {
	h := &hotkeyController{registry: app.GlobalShortcut, store: store, action: action}
	type request struct {
		Kind string           `json:"kind"`
		Keys settings.Hotkeys `json:"keys"`
	}
	queue := make(chan request, 32)
	enqueue := func(kind string) { queue <- request{Kind: kind} }
	go func() {
		for r := range queue {
			err := h.handle(r.Kind, r.Keys)
			message := ""
			if err != nil {
				message = err.Error()
			}
			data, loadErr := store.Load()
			if loadErr != nil {
				message = loadErr.Error()
			}
			app.Event.Emit("settings:hotkeys", map[string]any{"keys": data.EffectiveHotkeys(), "recording": r.Kind == "record" && err == nil, "error": message})
		}
	}()
	app.Event.On("settings:hotkey", func(e *application.CustomEvent) {
		b, err := json.Marshal(e.Data)
		if err != nil {
			return
		}
		var r request
		if json.Unmarshal(b, &r) == nil {
			queue <- r
		}
	})
	app.Event.On("settings:ready", func(*application.CustomEvent) { enqueue("sync") })
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) { enqueue("sync") })
	window.OnWindowEvent(events.Common.WindowLostFocus, func(*application.WindowEvent) { enqueue("cancel") })
	window.OnWindowEvent(events.Common.WindowClosing, func(*application.WindowEvent) { enqueue("cancel") })
	return func() { enqueue("sync") }
}
