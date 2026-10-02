//go:build darwin && cgo

package main

import (
	"bytes"
	"embed"
	"flag"
	"image/color"
	"image/png"
	"io/fs"
	"log"
	"os"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"talkingduck/cli"
	"talkingduck/menubarimage"
)

//go:embed frontend/*
var frontend embed.FS

var sample = []menubarimage.Cell{
	{Number: 1, Stage: "L", State: "attention"}, {Number: 2, Stage: "H", State: "attention"},
	{Number: 3, Stage: "-", State: "idle"}, {Number: 4, Stage: "F", State: "attention"},
	{Number: 5, Stage: "F", State: "attention"}, {Number: 6, Stage: "R", State: "attention"},
	{Number: 7, Stage: "F", State: "unknown"}, {Number: 8, Stage: "L", State: "attention"},
	{Number: 9, Stage: "H", State: "unknown"}, {Number: 10, Stage: "H", State: "attention"},
}

func icon(cells []menubarimage.Cell, mode menubarimage.Mode) ([]byte, error) {
	im := menubarimage.Render(cells, color.NRGBA{A: 255}, mode)
	// Upscale the deliberately pixel-aligned font to a 2x representation.
	var b bytes.Buffer
	// Encode at 2x so AppKit has the full backing resolution at the renderer’s logical size.
	if err := png.Encode(&b, menubarimage.Scale(im, 2)); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func main() {
	log.SetFlags(0)
	opts := cli.RegisterFlags(flag.CommandLine)
	gui := flag.Bool("gui", os.Getenv("TALKINGDUCK_GUI") == "1", "show the mock menu bar app")
	flag.Parse()
	if !*gui {
		os.Exit(cli.Run(opts, flag.Args()))
	}
	if len(flag.Args()) > 0 {
		opts.Inventory.Paths = flag.Args()
	}
	if err := run(opts); err != nil {
		log.Fatal(err)
	}
}

func run(opts *cli.Options) error {
	assets, err := fs.Sub(frontend, "frontend")
	if err != nil {
		return err
	}
	app := application.New(application.Options{
		Name: "TalkingDuck", Description: "Mock checkout status experiment",
		Mac:    application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
	})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "TalkingDuck — Inventory", Width: 1280, Height: 720, URL: "/", Hidden: true,
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); window.Hide() })
	view := &inventoryView{app: app, options: opts.Inventory}
	tray := app.SystemTray.New()
	data, err := icon(sample, menubarimage.Standard)
	if err != nil {
		return err
	}
	tray.SetTemplateIcon(data)
	tray.OnClick(func() {
		if window.IsVisible() && window.IsFocused() {
			log.Print("tray: hide overview")
			window.Hide()
			return
		}
		log.Print("tray: show and focus overview")
		window.Show()
		window.Focus()
	})
	// Bypass Wails' pre-click menu attachment. Programmatic OpenMenu tracks the
	// menu explicitly and clears the temporary attachment after dismissal.
	tray.OnRightClick(func() { log.Print("tray: right-click open menu"); tray.OpenMenu() })

	var mu sync.Mutex
	cells := append([]menubarimage.Cell(nil), sample...)
	compact := false
	var compactItem *application.MenuItem
	update := func(action string) {
		mu.Lock()
		defer mu.Unlock()
		if action == "compact" {
			compact = !compact
		}
		mode, padding := menubarimage.Standard, 8
		if compact {
			mode, padding = menubarimage.Compact, 6
		}
		if compactItem != nil {
			compactItem.SetChecked(compact)
		}
		data, err := icon(cells, mode)
		bounds := menubarimage.Render(cells, color.NRGBA{A: 255}, mode).Bounds()
		if err == nil {
			err = updateWideTray(tray, data, bounds.Dx(), bounds.Dy(), padding)
		}
		if err != nil {
			log.Printf("tray update failed: %v", err)
			app.Event.Emit("mock:error", err.Error())
			return
		}
		log.Printf("tray: image=%dx%d points; item=%d points", bounds.Dx(), bounds.Dy(), bounds.Dx()+padding)
	}
	menu := app.Menu.New()
	menu.Add("TalkingDuck").SetEnabled(false)
	menu.Add("Open overview").OnClick(func(*application.Context) { window.Show(); window.Focus() })
	compactItem = menu.AddCheckbox("Compact menu bar", false).OnClick(func(*application.Context) { update("compact") })
	menu.Add("Refresh inventory").OnClick(func(*application.Context) { view.refresh() })
	menu.AddSeparator()
	menu.Add("Quit TalkingDuck").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)
	app.Event.On("inventory:ready", func(*application.CustomEvent) { view.publish() })
	app.Event.On("inventory:refresh", func(*application.CustomEvent) { view.refresh() })
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) { update(""); view.refresh() })
	return app.Run()
}
