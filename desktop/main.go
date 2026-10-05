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

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"talkingduck/cli"
	"talkingduck/inventory"
	"talkingduck/mcpserver"
	"talkingduck/menubarimage"
)

//go:embed frontend/*
var frontend embed.FS

func icon(cells []menubarimage.Cell, mode menubarimage.Mode) ([]byte, error) {
	im := menubarimage.RenderAgents(cells, color.NRGBA{A: 255}, mode)
	// Agent motifs use four backing pixels per logical point.
	var b bytes.Buffer
	// The tray adapter sets the logical image size explicitly.
	if err := png.Encode(&b, im); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func main() {
	log.SetFlags(0)
	opts := cli.RegisterFlags(flag.CommandLine)
	flag.Parse()
	mode, err := opts.Mode("gui")
	if err != nil {
		log.Print(err)
		os.Exit(2)
	}
	if mode == "tui" {
		os.Exit(cli.RunTUI(opts, flag.Args()))
	}
	if mode != "gui" {
		os.Exit(cli.Run(opts, flag.Args()))
	}
	if len(flag.Args()) > 0 {
		opts.Inventory.Paths = flag.Args()
	}
	if err := run(opts, opts.MCPPort); err != nil {
		log.Fatal(err)
	}
}

func run(opts *cli.Options, mcpPort int) error {
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
	var settingsWindow *application.WebviewWindow
	showSettings := func() { settingsWindow.Show(); settingsWindow.Focus() }
	app := application.New(application.Options{
		KeyBindings: map[string]func(application.Window){"CmdOrCtrl+,": func(application.Window) { showSettings() }},
		Name:        "TalkingDuck", Description: "Checkout inventory", OnShutdown: cancel,
		Mac:    application.MacOptions{ActivationPolicy: application.ActivationPolicyAccessory},
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
	})
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "TalkingDuck — Inventory", Width: 1280, Height: 720, URL: "/", Hidden: true,
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); window.Hide() })
	settingsWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "TalkingDuck — Settings", Width: 520, Height: 250, URL: "/settings.html", Hidden: true,
	})
	settingsWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); settingsWindow.Hide() })
	var settingsMu sync.Mutex
	mcpAddress, mcpError := "", "Starting server…"
	publishSettings := func() {
		settingsMu.Lock()
		defer settingsMu.Unlock()
		app.Event.Emit("settings:snapshot", map[string]string{"address": mcpAddress, "error": mcpError})
	}
	app.Event.On("settings:ready", func(*application.CustomEvent) { publishSettings() })
	app.Event.On("settings:open", func(*application.CustomEvent) { showSettings() })
	view := &inventoryView{cache: cache, fetch: fetchRemote, emit: func(state inventoryState) { app.Event.Emit("inventory:snapshot", state) }}
	view.startQueues(ctx, !opts.Inventory.NoAgents)
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
		bounds := menubarimage.RenderAgents(cells, color.NRGBA{A: 255}, mode).Bounds()
		if err == nil {
			err = updateWideTray(tray, data, bounds.Dx()/4, bounds.Dy()/4, padding)
		}
		if err != nil {
			log.Printf("tray update failed: %v", err)
			app.Event.Emit("mock:error", err.Error())
			return
		}
		log.Printf("tray: image=%dx%d points; item=%d points", bounds.Dx()/4, bounds.Dy()/4, bounds.Dx()/4+padding)
	}
	view.updateTray = func(snapshot *inventory.Snapshot) { update("", snapshot) }
	menu := app.Menu.New()
	menu.Add("TalkingDuck").SetEnabled(false)
	menu.Add("Open overview").OnClick(func(*application.Context) { window.Show(); window.Focus() })
	compactItem = menu.AddCheckbox("Compact menu bar", false).OnClick(func(*application.Context) { update("compact", nil) })
	menu.Add("Refresh inventory").OnClick(func(*application.Context) { view.refresh() })
	menu.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) { showSettings() })
	menu.AddSeparator()
	menu.Add("Quit TalkingDuck").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)
	app.Event.On("inventory:ready", func(*application.CustomEvent) { view.publish() })
	app.Event.On("inventory:open-agent", func(event *application.CustomEvent) {
		link, ok := event.Data.(string)
		if !ok {
			return
		}
		go func() {
			if err := view.openAgent(link); err != nil {
				app.Event.Emit("inventory:open-error", err.Error())
			}
		}()
	})
	app.Event.On("inventory:open-ticket", func(event *application.CustomEvent) {
		ticket, ok := event.Data.(string)
		if !ok {
			return
		}
		go func() {
			if err := view.openTicket(ticket); err != nil {
				app.Event.Emit("inventory:open-error", err.Error())
			}
		}()
	})
	app.Event.On("inventory:open-folder", func(event *application.CustomEvent) {
		path, ok := event.Data.(string)
		if !ok {
			return
		}
		go func() {
			if err := view.openFolder(path); err != nil {
				app.Event.Emit("inventory:open-error", err.Error())
			}
		}()
	})
	app.Event.On("inventory:open-spec", func(event *application.CustomEvent) {
		path, ok := event.Data.(string)
		if !ok {
			return
		}
		go func() {
			if err := view.openSpec(path); err != nil {
				app.Event.Emit("inventory:open-error", err.Error())
			}
		}()
	})
	app.Event.On("inventory:refresh", func(*application.CustomEvent) { view.refresh() })
	app.Event.On("inventory:archive-row", func(event *application.CustomEvent) {
		data, ok := event.Data.(map[string]any)
		if !ok {
			return
		}
		folder, folderOK := data["folder"].(string)
		spec, specOK := data["spec"].(string)
		if !folderOK || !specOK {
			return
		}
		go func() {
			if err := view.archiveRow(folder, spec); err != nil {
				app.Event.Emit("inventory:open-error", err.Error())
			} else {
				app.Event.Emit("inventory:archived")
			}
		}()
	})

	app.Event.On("inventory:refresh-folder", func(event *application.CustomEvent) {
		path, ok := event.Data.(string)
		if !ok {
			return
		}
		go func() {
			if err := view.RefreshFolder(ctx, path); err != nil {
				app.Event.Emit("inventory:open-error", err.Error())
			}
		}()
	})

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		if err := installTrayMenu(tray); err != nil {
			log.Printf("tray menu setup failed: %v", err)
		}
		update("", nil)
		server, addr, err := mcpserver.Start(mcpPort, view)
		settingsMu.Lock()
		if err != nil {
			mcpError = "Server unavailable: " + err.Error()
			log.Printf("MCP server failed: %v", err)
			app.Event.Emit("mock:error", "MCP server failed: "+err.Error())
		} else {
			mcpAddress, mcpError = "http://"+addr.String()+"/mcp", ""
			log.Printf("MCP listening at http://%s/mcp", addr)
			go func() { <-ctx.Done(); _ = server.Close() }()
		}
		settingsMu.Unlock()
		publishSettings()
		go view.watchChanges(ctx, !opts.Inventory.NoAgents)
	})
	return app.Run()
}
