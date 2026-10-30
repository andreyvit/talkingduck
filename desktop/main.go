//go:build darwin && cgo

package main

import (
	"bytes"
	"context"
	"embed"
	"flag"
	"image"
	"image/color"
	"io/fs"
	"log"
	"os"
	"reflect"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"talkingduck/cli"
	"talkingduck/inventory"
	"talkingduck/mcpserver"
	"talkingduck/menubarimage"
	"talkingduck/settings"
	"talkingduck/telemetry"
	"time"
)

//go:embed frontend/*
var frontend embed.FS

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
	store := &settings.Store{Path: opts.SettingsPath}
	if store.Path == "" {
		store.Path = settings.DefaultPath()
	}
	config, err := opts.CollectionOptions(nil)
	if err != nil {
		return err
	}
	cache, err := inventory.New(config)
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
		Title: "Zoo Agents Dashboard", Width: 1280, Height: 720, URL: "/", Hidden: true,
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); window.Hide() })
	settingsWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "TalkingDuck — Settings", Width: 880, Height: 620, MinWidth: 740, MinHeight: 540, URL: "/settings.html", Hidden: true,
	})
	settingsWindow.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); settingsWindow.Hide() })
	for name, w := range map[string]*application.WebviewWindow{"Inventory": window, "Settings": settingsWindow} {
		var once sync.Once
		w.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
			once.Do(func() {
				if err := rememberWindowFrame(w, name); err != nil {
					log.Print(err)
				}
			})
		})
	}
	var view *inventoryView
	syncHotkeys := func() {}
	syncMonitor := func() {}
	var settingsMu sync.Mutex
	mcpAddress, mcpError := "", "Starting server…"
	publishSettings := func() {
		settingsMu.Lock()
		defer settingsMu.Unlock()
		syncHotkeys()
		syncMonitor()
		if energyMask.Load()&8 == 0 {
			return
		}
		folders, err := view.ListFolders()
		settingsError := ""
		if err != nil {
			settingsError = err.Error()
		}
		preferences, _ := store.Load()
		publishArchivedSettings(app, view)
		app.Event.Emit("settings:snapshot", map[string]any{"menubar": preferences.Menubar.Defaults(), "applications": applicationSettings(view), "address": mcpAddress, "error": mcpError, "folders": folders, "settingsError": settingsError, "path": store.Path})
	}
	app.Event.On("settings:ready", func(*application.CustomEvent) { publishSettings() })
	app.Event.On("settings:open", func(*application.CustomEvent) { showSettings() })
	view = &inventoryView{folderOverride: len(opts.Inventory.Paths) > 0, settings: store, settingsChanged: publishSettings, settingsError: func(err error) { app.Event.Emit("settings:error", err.Error()) }, cache: cache, fetch: fetchRemote, emit: func(state inventoryState) {
		if energyMask.Load()&2 != 0 {
			app.Event.Emit("inventory:snapshot", state)
		}
	}}

	view.markdown = newMarkdownViewer(ctx, app, window, store)
	view.markdown.title = view.markdownTitle
	registerMarkdownOpen(app, view.markdown)
	app.Event.On("inventory:preview", func(*application.CustomEvent) { go view.markdown.showLast() })
	registerApplicationSettings(app, view, publishSettings)

	app.Event.On("settings:visibility", func(event *application.CustomEvent) {
		data, ok := event.Data.(map[string]any)
		if !ok {
			return
		}
		path, _ := data["path"].(string)
		visible, ok := data["visible"].(bool)
		if !ok {
			return
		}
		go func() {
			var err error
			if visible {
				err = view.AddMonitoredFolder(path)
			} else {
				err = view.RemoveMonitoredFolder(path)
			}
			if err != nil {
				view.settingsError(err)
				publishSettings()
			}
		}()
	})
	app.Event.On("settings:shortcut", func(event *application.CustomEvent) {
		data, ok := event.Data.(map[string]any)
		if !ok {
			return
		}
		path, _ := data["path"].(string)
		shortcut, ok := data["shortcut"].(string)
		if !ok {
			return
		}
		go func() {
			if err := view.UpdateFolderMenubarItemShortcut(path, shortcut); err != nil {
				view.settingsError(err)
				publishSettings()
			}
		}()
	})
	app.Event.On("settings:add-folder", func(*application.CustomEvent) {
		go func() {
			path, err := app.Dialog.OpenFile().CanChooseFiles(false).CanChooseDirectories(true).PromptForSingleSelection()
			if err == nil && path != "" {
				err = view.AddMonitoredFolder(path)
			}
			if err != nil {
				view.settingsError(err)
			}
		}()
	})
	monitorChanges := make(chan struct{}, 1)
	signalMonitor := func() {
		select {
		case monitorChanges <- struct{}{}:
		default:
		}
	}
	initialSettings, _ := store.Load()
	monitoring := telemetry.NewSuspended(monitorConfig(initialSettings.Menubar), func(telemetry.Snapshot) { signalMonitor() })
	defer monitoring.Close()
	publishMonitor := func() {
		if energyMask.Load()&2 == 0 {
			return
		}
		preferences, _ := store.Load()
		app.Event.Emit("monitor:snapshot", map[string]any{"snapshot": monitoring.Snapshot(), "config": preferences.Menubar.Defaults()})
	}
	syncMonitor = func() {
		preferences, err := store.Load()
		if err == nil {
			monitoring.Update(monitorConfig(preferences.Menubar))
			signalMonitor()
		}
	}
	registerMenubarSettings(app, store, func() { publishSettings() })
	app.Event.On("energy:ready", func(*application.CustomEvent) {
		mask := energyMask.Load()
		app.Event.Emit("energy:visibility", map[string]bool{"dashboard": mask&2 != 0, "markdown": mask&4 != 0})
		view.publish()
		publishMonitor()
	})
	app.Event.On("monitor:ready", func(*application.CustomEvent) { publishMonitor() })
	view.startQueues(ctx, !opts.Inventory.NoAgents)
	view.cheap.SetPaused(true)
	view.expensive.SetPaused(true)
	trayCells := func(snapshot *inventory.Snapshot) []menubarimage.Cell {
		data, err := store.Load()
		if err != nil {
			return nil
		}
		return inventoryCells(snapshot, data.Folders)
	}
	cells := trayCells(cache.Snapshot())
	tray := app.SystemTray.New()
	// The first native update supplies the image after startup. Never enqueue
	// a Wails image setter that can later overwrite the atomic native update.

	toggleOverview := func() {
		if window.IsVisible() && window.IsFocused() {
			log.Print("tray: hide overview")
			window.Hide()
			return
		}
		log.Print("tray: show and focus overview")
		window.Show()
		window.Focus()
	}
	tray.OnClick(toggleOverview)
	// Disable Wails' automatic menu attachment. The native monitor installed
	// after startup handles right-clicks before the status button tracks them.
	tray.OnRightClick(func() {})

	var mu sync.Mutex
	compact := false
	var compactItem *application.MenuItem
	var lastTray []byte
	var lastInputs any
	lastPadding := -1
	update := func(action string, snapshot *inventory.Snapshot) {
		mu.Lock()
		defer mu.Unlock()
		if snapshot != nil {
			cells = trayCells(snapshot)
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
		preferences, loadErr := store.Load()
		if loadErr != nil {
			return
		}
		menuConfig := preferences.Menubar.Defaults()
		art := monitorArt(monitoring.Snapshot(), menuConfig, time.Now())
		art.Mode = mode
		if energyMask.Load()&1 == 0 {
			return
		}
		inputs := struct {
			Art      menubarimage.Monitor
			Cells    []menubarimage.Cell
			Projects string
			Padding  int
		}{art, append([]menubarimage.Cell(nil), cells...), menuConfig.Projects, padding}
		if reflect.DeepEqual(lastInputs, inputs) {
			return
		}
		prefix := menubarimage.RenderMonitor(art)
		var projects *image.NRGBA
		if menuConfig.Projects != "off" {
			projects = trayImage(cells, mode)
		}
		if prefix.Bounds().Empty() && projects == nil {
			projects = trayImage(nil, mode)
		}
		im := menubarimage.JoinMonitor(prefix, projects)
		var encoded bytes.Buffer
		err := menubarimage.EncodePNG(&encoded, im)
		data := encoded.Bytes()
		bounds := im.Bounds()
		if err == nil && bytes.Equal(lastTray, data) && lastPadding == padding {
			lastInputs = inputs
			return
		}
		if err == nil {
			err = updateWideTray(tray, data, bounds.Dx()/4, bounds.Dy()/4, padding, trayMonitorStyle{Width: prefix.Bounds().Dx() / 4, TintPing: false})
		}
		if err != nil {
			log.Printf("tray update failed: %v", err)
			app.Event.Emit("mock:error", err.Error())
			return
		}
		lastInputs = inputs
		lastTray = append(lastTray[:0], data...)
		lastPadding = padding
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

	syncHotkeys = registerHotkeySettings(app, settingsWindow, store, toggleOverview, func() { go view.markdown.show() })

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		// Older Settings frames predate the folder list.
		width, height := settingsWindow.Size()
		if width < 740 || height < 540 {
			settingsWindow.SetSize(max(width, 740), max(height, 540))
		}
		if err := installTrayMenu(tray); err != nil {
			log.Printf("tray menu setup failed: %v", err)
		}
		update("", nil)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-monitorChanges:
					publishMonitor()
					update("", nil)
				}
			}
		}()
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
		go observeEnergy(ctx, app, window, settingsWindow, view, monitoring, !opts.Inventory.NoAgents, func() { view.publish(); publishMonitor(); publishSettings(); update("", nil) })
	})
	return app.Run()
}

// Keep Settings reachable even when no folder belongs in the menu bar.
func trayImage(cells []menubarimage.Cell, mode menubarimage.Mode) *image.NRGBA {
	if len(cells) == 0 {
		return menubarimage.RenderAgents([]menubarimage.Cell{{Shortcut: "?", Stage: "", State: "idle"}}, color.NRGBA{A: 255}, mode)
	}
	return menubarimage.RenderAgents(cells, color.NRGBA{A: 255}, mode)
}
