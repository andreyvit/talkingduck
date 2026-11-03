//go:build darwin && cgo

package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	html "golang.org/x/net/html"
)

type previewImage struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

func documentImage(markup string, index int) (previewImage, bool) {
	if index < 0 {
		return previewImage{}, false
	}
	tokens := html.NewTokenizer(strings.NewReader(markup))
	for {
		kind := tokens.Next()
		if kind == html.ErrorToken {
			return previewImage{}, false
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		name, hasAttrs := tokens.TagName()
		if string(name) != "img" {
			continue
		}
		if index > 0 {
			index--
			continue
		}
		var img previewImage
		for hasAttrs {
			key, val, more := tokens.TagAttr()
			hasAttrs = more
			switch string(key) {
			case "src":
				img.Src = string(val)
			case "alt":
				img.Alt = string(val)
			}
		}
		return img, img.Src != ""
	}
}

func installImagePreview(v *markdownViewer) {
	w := v.app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "Image Preview", Width: 960, Height: 720, Hidden: true, URL: "/image-preview.html"})
	var mu sync.Mutex
	var current previewImage
	var ready bool
	publish := func() {
		if ready && current.Src != "" {
			data, _ := json.Marshal(current)
			w.ExecJS("window.showImage(" + string(data) + ")")
		}
	}
	open := func(key string) {
		gen, idx, ok := strings.Cut(key, "/")
		if !ok {
			return
		}
		generation, err := strconv.ParseUint(gen, 10, 64)
		if err != nil {
			return
		}
		index, err := strconv.Atoi(idx)
		if err != nil {
			return
		}
		v.mu.Lock()
		if generation != v.generation {
			v.mu.Unlock()
			return
		}
		markup := v.document.HTML
		v.mu.Unlock()
		img, ok := documentImage(markup, index)
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
		current = previewImage{}
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
	menu := v.app.ContextMenu.New()
	menu.Add("Open Image in Separate Window").OnClick(func(c *application.Context) { key := c.ContextMenuData(); go open(key) })
	v.app.ContextMenu.Add("markdown-image", menu)
}
