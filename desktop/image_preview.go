//go:build darwin && cgo

package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"talkingduck/markdownview"
)

func installImagePreview(v *markdownViewer) {
	w := v.app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "Image Preview", Width: 960, Height: 720, Hidden: true, URL: "/image-preview.html"})
	var mu sync.Mutex
	var current markdownview.Image
	var ready bool
	publish := func() {
		if ready && current.Src != "" {
			data, _ := json.Marshal(current)
			w.ExecJS("window.showImage(" + string(data) + ")")
		}
	}
	resolve := func(key string) (markdownview.Image, bool) {
		if key == "preview" {
			mu.Lock()
			defer mu.Unlock()
			return current, current.Src != ""
		}
		gen, idx, ok := strings.Cut(strings.TrimSpace(key), "/")
		generation, e1 := strconv.ParseUint(gen, 10, 64)
		index, e2 := strconv.Atoi(idx)
		v.mu.Lock()
		defer v.mu.Unlock()
		if !ok || e1 != nil || e2 != nil || generation != v.generation || index < 0 || index >= len(v.document.Images) {
			return markdownview.Image{}, false
		}
		img := v.document.Images[index]
		return img, img.Src != ""
	}
	open := func(key string) {
		img, ok := resolve(key)
		if !ok {
			return
		}
		mu.Lock()
		current = img
		publish()
		mu.Unlock()
		title := img.Alt
		if title == "" {
			title = "Image Preview"
		}
		w.SetTitle(title)
		w.UnMinimise()
		placeImagePreview(w, v.window)
		w.Show()
		w.Focus()
	}
	closePreview := func() {
		w.Hide()
		mu.Lock()
		current = markdownview.Image{}
		if ready {
			w.ExecJS("window.clearImage()")
		}
		mu.Unlock()
		v.window.Show()
		v.window.Focus()
	}
	w.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); go closePreview() })
	v.app.Event.On("image:close", func(*application.CustomEvent) { go closePreview() })
	v.app.Event.On("image:ready", func(*application.CustomEvent) { mu.Lock(); ready = true; publish(); mu.Unlock() })
	v.app.Event.On("image:open", func(e *application.CustomEvent) {
		if key, ok := e.Data.(string); ok {
			go open(key)
		}
	})
	register := func(kind, format string) {
		menu := v.app.ContextMenu.New()
		action := func(label string, fn func(markdownview.Image) error) {
			menu.Add(label).OnClick(func(c *application.Context) {
				img, ok := resolve(c.ContextMenuData())
				if !ok {
					return
				}
				go func() {
					if err := fn(img); err != nil {
						v.app.Dialog.Error().SetTitle(label).SetMessage(err.Error()).Show()
					}
				}()
			})
		}
		copyText := func(text string) error {
			if !v.app.Clipboard.SetText(text) {
				return fmt.Errorf("could not write clipboard")
			}
			return nil
		}
		menu.Add("Open Image in Separate Window").OnClick(func(c *application.Context) { key := c.ContextMenuData(); go open(key) })
		if kind == "file" {
			action("Open", func(i markdownview.Image) error { return openImageFile(i.Path, false) })
			action("Reveal in Finder", func(i markdownview.Image) error { return openImageFile(i.Path, true) })
			action("Copy Path", func(i markdownview.Image) error { return copyText(i.Path) })
		}
		if kind == "mermaid" {
			action("Copy Mermaid Source", func(i markdownview.Image) error { return copyText(i.Mermaid) })
		}
		action("Copy Image as "+format, func(i markdownview.Image) error { return copyImage(i, format) })
		if format != "PNG" {
			action("Copy Image as PNG", func(i markdownview.Image) error { return copyImage(i, "PNG") })
		}
		v.app.ContextMenu.Add("image-"+kind+"-"+format, menu)
	}
	register("mermaid", "SVG")
	for _, f := range markdownview.ImageFormats {
		register("file", f)
		register("embedded", f)
	}
}
