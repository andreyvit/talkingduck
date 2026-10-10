//go:build darwin && cgo

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"talkingduck/markdownview"
	"talkingduck/settings"
	"talkingduck/table"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

type markdownViewer struct {
	mu           sync.Mutex
	app          *application.App
	window, main *application.WebviewWindow
	store        *settings.Store
	history      markdownview.History
	historyPath  string
	document     markdownview.Document
	refresh      markdownview.Refresh
	stamp        string
	generation   uint64
	visible      bool
	title        func(string) string
}

func newMarkdownViewer(ctx context.Context, app *application.App, main *application.WebviewWindow, store *settings.Store) *markdownViewer {
	v := &markdownViewer{app: app, main: main, store: store, historyPath: store.Path + ".markdown-history.json"}
	var err error
	v.history, err = markdownview.LoadHistory(v.historyPath)
	if err != nil {
		log.Printf("Markdown history: %v", err)
	}
	v.window = app.Window.NewWithOptions(application.WebviewWindowOptions{Title: "Markdown Preview", Width: 1060, Height: 780, MinWidth: 540, MinHeight: 320, URL: "/markdown.html", Hidden: true})
	var once sync.Once
	v.window.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		once.Do(func() {
			if err := rememberWindowFrame(v.window, "Markdown"); err != nil {
				log.Print(err)
			}
		})
	})
	v.window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); go v.close() })
	app.Event.On("markdown:ready", func(*application.CustomEvent) { go func() { v.mu.Lock(); defer v.mu.Unlock(); v.publish("") }() })
	app.Event.On("markdown:command", func(e *application.CustomEvent) {
		var req struct {
			Key      string                `json:"key"`
			Shift    bool                  `json:"shift"`
			Modified bool                  `json:"modified"`
			Action   string                `json:"action"`
			Path     string                `json:"path"`
			Position markdownview.Position `json:"position"`
			Heading  int                   `json:"heading"`
			Link     string                `json:"link"`
		}
		b, _ := json.Marshal(e.Data)
		if json.Unmarshal(b, &req) != nil {
			return
		}
		go func() {
			v.mu.Lock()
			defer v.mu.Unlock()
			if req.Path != "" {
				v.history.Remember(req.Path, req.Position)
			}
			action := req.Action
			if req.Key != "" {
				action = markdownview.KeyAction(req.Key, req.Shift, req.Modified)
			}
			if action == "position" {
				v.save()
				return
			}
			if req.Path != "" && req.Path != v.history.Current() {
				return
			}
			switch action {
			case "dashboard":
				v.save()
				v.main.UnMinimise()
				v.main.Show()
				v.main.Focus()
			case "history":
				v.app.Event.Emit("markdown:history", v.history.Recent)
			case "recent":
				known := false
				for _, visit := range v.history.Recent {
					if visit.Path == req.Link {
						known = true
						break
					}
				}
				if known {
					v.history.Open(req.Link)
					v.selectDashboardRow()
					v.render()
					v.save()
				}
			case "close":
				v.closeLocked()
			case "refresh":
				v.render()
			case "back", "forward":
				delta := -1
				if action == "forward" {
					delta = 1
				}
				if v.history.Move(delta) {
					v.selectDashboardRow()
					v.render()
					v.save()
				}
			case "toc":
				v.history.HideTOC = !v.history.HideTOC
				v.save()
				v.app.Event.Emit("markdown:effect", map[string]any{"action": "toc", "hidden": v.history.HideTOC})
			case "heading-up", "heading-down":
				delta := -1
				if action == "heading-down" {
					delta = 1
				}
				i := markdownview.NextHeading(req.Heading, delta, len(v.document.Headings))
				if i >= 0 {
					v.app.Event.Emit("markdown:effect", map[string]any{"action": "heading", "id": v.document.Headings[i].ID})
				}
			case "page-up", "page-down":
				v.app.Event.Emit("markdown:effect", map[string]any{"action": action})
			case "link":
				v.link(req.Link)
			}
		}()
	})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				v.mu.Lock()
				v.save()
				v.mu.Unlock()
				return
			case now := <-ticker.C:
				v.mu.Lock()
				if v.visible && v.history.Current() != "" {
					stamp := fileStamp(v.history.Current())
					if stamp != v.stamp {
						v.stamp = stamp
						v.refresh.Changed(now)
					}
					if v.refresh.Due(now) {
						v.render()
					}
				}
				v.mu.Unlock()
			}
		}
	}()
	return v
}
func fileStamp(path string) string {
	info, e := os.Stat(path)
	if e != nil {
		return e.Error()
	}
	return fmt.Sprintf("%d:%d:%v", info.Size(), info.ModTime().UnixNano(), info.Sys().(*syscall.Stat_t).Ino)
}
func (v *markdownViewer) open(path string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, err := markdownview.ReadFile(path, markdownview.MaxDocument); err != nil {
		return err
	}
	v.history.Open(path)
	v.visible = true
	v.render()
	v.save()
	v.window.UnMinimise()
	v.window.Show()
	v.window.Focus()
	return nil
}
func (v *markdownViewer) save() {
	if err := v.history.Save(v.historyPath); err != nil {
		v.app.Event.Emit("markdown:error", err.Error())
	}
}
func (v *markdownViewer) close() { v.mu.Lock(); defer v.mu.Unlock(); v.closeLocked() }
func (v *markdownViewer) closeLocked() {
	v.visible = false
	v.save()
	v.window.Hide()
	v.main.Show()
	v.main.Focus()
}
func (v *markdownViewer) render() {
	path := v.history.Current()
	if path == "" {
		return
	}
	v.stamp = fileStamp(path)
	source, err := markdownview.ReadFile(path, markdownview.MaxDocument)
	var d markdownview.Document
	if err == nil {
		d, err = markdownview.Render(source, path)
	}
	v.refresh.Complete(time.Now())
	v.generation++
	if err != nil {
		v.document = markdownview.Document{Headings: []markdownview.Heading{}}
		v.publish(err.Error())
		return
	}
	v.document = d
	title := markdownWindowTitle(path, table.Table{})
	if v.title != nil {
		title = v.title(path)
	}
	v.window.SetTitle(title)
	v.publish("")
}
func (v *markdownViewer) publish(message string) {
	d, err := v.store.Load()
	if err != nil {
		message = err.Error()
	}
	v.app.Event.Emit("markdown:snapshot", map[string]any{"document": v.document, "path": v.history.Current(), "position": v.history.Position(v.history.Current()), "hideTOC": v.history.HideTOC, "back": v.history.Index > 0, "forward": v.history.Index+1 < len(v.history.Visits), "appearance": d.MarkdownAppearance.Defaults(), "generation": v.generation, "error": message})
}
func (v *markdownViewer) link(raw string) {
	u, err := url.Parse(raw)
	if err != nil {
		return
	}
	if u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "mailto" {
		v.app.Browser.OpenURL(raw)
		return
	}
	if (u.Scheme != "" && u.Scheme != "file") || u.Host != "" {
		return
	}
	path := u.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(v.history.Current()), path)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown":
	default:
		v.app.Event.Emit("markdown:error", "Link is not a Markdown file")
		return
	}
	if _, err := markdownview.ReadFile(path, markdownview.MaxDocument); err != nil {
		v.app.Event.Emit("markdown:error", err.Error())
		return
	}
	v.history.Open(filepath.Clean(path))
	if u.Fragment != "" {
		v.history.Remember(v.history.Current(), markdownview.Position{Heading: u.Fragment})
	}
	v.render()
	v.save()
}

func (v *markdownViewer) show() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.visible = true
	if v.history.Current() != "" {
		v.render()
	} else {
		v.app.Event.Emit("markdown:history", v.history.Recent)
	}
	v.window.UnMinimise()
	v.window.Show()
	v.window.Focus()
}

// Use the same folder label and visible rows as the dashboard. Nested roots win.
func markdownWindowTitle(path string, model table.Table) string {
	parent := filepath.Dir(path)
	best := ""
	column := -1
	for i, c := range model.Columns {
		if c.ID == "folder" {
			column = i
			break
		}
	}
	if column >= 0 {
		for _, row := range model.Rows {
			root := filepath.Clean(row.Group)
			if root == "." || len(root) <= len(best) || column >= len(row.Cells) {
				continue
			}
			relative, err := filepath.Rel(root, path)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				continue
			}
			label := row.Cells[column].Text()
			if label != "" {
				best = root
				parent = label
			}
		}
	}
	return filepath.Base(path) + " — " + parent
}

func (v *inventoryView) markdownTitle(path string) string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return markdownWindowTitle(path, v.state.Model)
}

func (v *markdownViewer) updateTitle() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if path := v.history.Current(); path != "" && v.title != nil {
		v.window.SetTitle(v.title(path))
	}
}

func (v *markdownViewer) selectDashboardRow() {
	v.app.Event.Emit("inventory:select-spec", v.history.Current())
}

func (v *markdownViewer) showLast() {
	v.mu.Lock()
	hasFile := v.history.Current() != ""
	v.mu.Unlock()
	if hasFile {
		v.show()
	}
}
