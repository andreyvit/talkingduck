//go:build darwin && cgo

package main

import (
	"context"
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"talkingduck/inventory"
)

type archiveCache interface {
	RecentArchives() ([]inventory.ArchivedRow, error)
	UnarchiveRow(string, string) error
}

func publishArchivedSettings(app *application.App, v *inventoryView) {
	cache, ok := v.cache.(archiveCache)
	if !ok {
		return
	}
	rows, err := cache.RecentArchives()
	if err != nil {
		v.settingsError(err)
		return
	}
	app.Event.Emit("settings:archives", rows)
}
func (v *inventoryView) unarchiveRow(folder, spec string) error {
	cache, ok := v.cache.(archiveCache)
	if !ok {
		return fmt.Errorf("unarchiving is unavailable")
	}
	if err := cache.UnarchiveRow(folder, spec); err != nil {
		return err
	}
	if v.settingsChanged != nil {
		v.settingsChanged()
	}
	for _, path := range v.cache.Paths() {
		if path == folder {
			return v.RefreshFolder(context.Background(), folder)
		}
	}
	return nil
}
func registerArchivedSettings(app *application.App, v *inventoryView) {
	app.Event.On("settings:unarchive", func(e *application.CustomEvent) {
		data, ok := e.Data.(map[string]any)
		if !ok {
			return
		}
		folder, ok := data["folder"].(string)
		if !ok {
			return
		}
		spec, ok := data["spec"].(string)
		if !ok {
			return
		}
		go func() {
			if err := v.unarchiveRow(folder, spec); err != nil {
				v.settingsError(err)
			}
		}()
	})
}
