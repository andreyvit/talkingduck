//go:build darwin && cgo

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func markdownFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown":
		return true
	}
	return false
}
func registerMarkdownOpen(app *application.App, v *markdownViewer) {
	app.Event.OnApplicationEvent(events.Common.ApplicationOpenedWithFile, func(e *application.ApplicationEvent) {
		path := e.Context().Filename()
		go func() {
			var err error
			if !markdownFile(path) {
				err = fmt.Errorf("not a Markdown file: %s", path)
			} else {
				err = v.open(path)
			}
			if err != nil {
				app.Event.Emit("markdown:error", err.Error())
				app.Event.Emit("inventory:open-error", err.Error())
			}
		}()
	})
}
