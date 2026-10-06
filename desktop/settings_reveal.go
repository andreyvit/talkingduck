//go:build darwin && cgo

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"talkingduck/launchers"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func settingsRevealCommand(path string) (launchers.Command, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return launchers.Command{}, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return launchers.Command{}, fmt.Errorf("configuration file has not been created yet; change a setting first")
	}
	if err != nil {
		return launchers.Command{}, err
	}
	if info.IsDir() {
		return launchers.Command{}, fmt.Errorf("configuration path is a directory")
	}
	return launchers.Command{Executable: "/usr/bin/open", Args: []string{"-R", path}}, nil
}
func registerSettingsReveal(app *application.App, v *inventoryView) {
	app.Event.On("settings:reveal-file", func(*application.CustomEvent) {
		go func() {
			cmd, err := settingsRevealCommand(v.settings.Path)
			if err == nil {
				err = cmd.Start()
			}
			if err != nil {
				v.settingsError(err)
			}
		}()
	})
}
