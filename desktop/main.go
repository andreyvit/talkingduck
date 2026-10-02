//go:build darwin && cgo

package main

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"image/color"
	"image/png"
	"io/fs"
	"log"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"talkingduck/cli"
	"talkingduck/inventory"
	"talkingduck/menubarimage"
)

//go:embed frontend/*
var frontend embed.FS

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
	gui := flag.Bool("gui", os.Getenv("TALKINGDUCK_GUI") == "1", "show the menu bar app")
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
	cache, err := inventory.New(opts.Inventory)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := application.New(application.Options{
		Name: "TalkingDuck", Description: "Checkout inventory", OnShutdown: cancel,
		Mac:    application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
	})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "TalkingDuck — Inventory", Width: 1280, Height: 720, URL: "/", Hidden: true,
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); window.Hide() })
	view := &inventoryView{cache: cache, requests: make(chan struct{}), emit: func(state inventoryState) { app.Event.Emit("inventory:snapshot", state) }}
	cells := inventoryCells(cache.Snapshot())
	tray := app.SystemTray.New()
	data, err := icon(cells, menubarimage.Standard)
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
	// Disable Wails' automatic menu attachment. The native monitor installed
	// after startup handles right-clicks before the status button tracks them.
	tray.OnRightClick(func() {})

	var mu sync.Mutex
	compact := false
	var compactItem *application.MenuItem
	update := func(action string, snapshot *inventory.Snapshot) {
		mu.Lock()
		defer mu.Unlock()
		if snapshot != nil {
			cells = inventoryCells(snapshot)
		}
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
	view.updateTray = func(snapshot *inventory.Snapshot) { update("", snapshot) }
	menu := app.Menu.New()
	menu.Add("TalkingDuck").SetEnabled(false)
	menu.Add("Open overview").OnClick(func(*application.Context) { window.Show(); window.Focus() })
	compactItem = menu.AddCheckbox("Compact menu bar", false).OnClick(func(*application.Context) { update("compact", nil) })
	menu.Add("Refresh inventory").OnClick(func(*application.Context) { view.refresh() })
	menu.AddSeparator()
	menu.Add("Quit TalkingDuck").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)
	app.Event.On("inventory:ready", func(*application.CustomEvent) { view.publish() })
	app.Event.On("inventory:refresh", func(*application.CustomEvent) { view.refresh() })
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if err := installTrayMenu(tray); err != nil {
			log.Printf("tray menu setup failed: %v", err)
		}
		update("", nil)
		go view.run(ctx, time.Minute)
	})
	return app.Run()
}
