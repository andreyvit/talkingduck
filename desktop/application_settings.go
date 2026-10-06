//go:build darwin && cgo

package main

import (
	"encoding/json"
	"runtime"
	"sync"
	"talkingduck/launchers"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func applicationSettings(v *inventoryView) map[string]any {
	data, err := v.settings.Load()
	message := ""
	if err != nil {
		message = err.Error()
	}
	return map[string]any{"values": data.Applications.Defaults(), "editors": launchers.Available(launchers.Editors, runtime.GOOS), "gitClients": launchers.Available(launchers.GitClients, runtime.GOOS), "trackers": launchers.Trackers, "error": message}
}
func registerApplicationSettings(app *application.App, v *inventoryView, publish func()) {
	registerSettingsReveal(app, v)
	var saveMu sync.Mutex
	latest := map[string]int{}
	app.Event.On("settings:application", func(e *application.CustomEvent) {
		var request struct {
			Client    string          `json:"client"`
			Kind      string          `json:"kind"`
			Revision  int             `json:"revision"`
			Selection json.RawMessage `json:"selection"`
		}
		raw, err := json.Marshal(e.Data)
		if err != nil {
			return
		}
		if json.Unmarshal(raw, &request) != nil {
			return
		}
		go func() {
			saveMu.Lock()
			defer saveMu.Unlock()
			key := request.Client + ":" + request.Kind
			if request.Revision < latest[key] {
				return
			}
			latest[key] = request.Revision
			var err error
			if request.Kind == "tracker" {
				var s launchers.TrackerSelection
				err = json.Unmarshal(request.Selection, &s)
				if err == nil {
					_, err = v.settings.SetTracker(s)
				}
			} else {
				var s launchers.Selection
				err = json.Unmarshal(request.Selection, &s)
				if err == nil {
					_, err = v.settings.SetApplication(request.Kind, s)
				}
			}
			message := ""
			if err != nil {
				message = err.Error()
			}
			app.Event.Emit("settings:application-result", map[string]any{"client": request.Client, "kind": request.Kind, "revision": request.Revision, "error": message})
			publish()
		}()
	})
	app.Event.On("settings:choose-application", func(e *application.CustomEvent) {
		kind, ok := e.Data.(string)
		if !ok || (kind != "editor" && kind != "git_client") {
			return
		}
		go func() {
			path, err := app.Dialog.OpenFile().CanChooseFiles(true).CanChooseDirectories(false).PromptForSingleSelection()
			if err != nil {
				v.settingsError(err)
				return
			}
			if path != "" {
				app.Event.Emit("settings:application-path", map[string]string{"kind": kind, "path": path})
			}
		}()
	})
	for _, kind := range []string{"editor", "git"} {
		app.Event.On("inventory:open-"+kind, func(e *application.CustomEvent) {
			path, ok := e.Data.(string)
			if !ok {
				return
			}
			go func() {
				if err := v.openProject(kind, path); err != nil {
					app.Event.Emit("inventory:open-error", err.Error())
				}
			}()
		})
	}
	app.Event.On("inventory:copy-ticket-url", func(e *application.CustomEvent) {
		ticket, ok := e.Data.(string)
		if !ok {
			return
		}
		go func() {
			if err := v.copyTicketURL(ticket); err != nil {
				app.Event.Emit("inventory:open-error", err.Error())
			} else {
				app.Event.Emit("inventory:notice", "Copied: "+ticket+" URL")
			}
		}()
	})
}
