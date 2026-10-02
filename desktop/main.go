//go:build darwin && cgo

package main

import (
	"bytes"
	"embed"
	"encoding/base64"
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
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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
		Title: "TalkingDuck — Mock overview", Width: 660, Height: 620, URL: "/", Hidden: true,
	})
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { e.Cancel(); window.Hide() })
	tray := app.SystemTray.New()
	data, err := icon(sample, menubarimage.Standard)
	if err != nil {
		return err
	}
	tray.SetTemplateIcon(data)
	tray.OnClick(func() { log.Print("tray: open overview"); window.Show(); window.Focus() })

	var mu sync.Mutex
	cells := append([]menubarimage.Cell(nil), sample...)
	styles := false
	compact := false
	var compactItem *application.MenuItem
	update := func(action string) {
		mu.Lock()
		defer mu.Unlock()
		if action == "styles" {
			styles = !styles
			cells = append([]menubarimage.Cell(nil), sample...)
			if styles {
				states := []string{"idle", "running", "attention", "unknown"}
				for i := range cells {
					cells[i].State = states[i%len(states)]
				}
			}
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
		log.Printf("tray: image=%dx%d points; item=%d points; style-sample=%t", bounds.Dx(), bounds.Dy(), bounds.Dx()+padding, styles)
		app.Event.Emit("mock:snapshot", map[string]any{"compact": compact, "width": bounds.Dx(), "height": bounds.Dy(), "cells": cells, "styles": styles, "image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)})
	}
	menu := app.Menu.New()
	menu.Add("TalkingDuck — mock data").SetEnabled(false)
	menu.Add("Open overview").OnClick(func(*application.Context) { window.Show(); window.Focus() })
	compactItem = menu.AddCheckbox("Compact menu bar", false).OnClick(func(*application.Context) { update("compact") })
	menu.Add("Toggle style sample").OnClick(func(*application.Context) { update("styles") })
	menu.Add("Refresh mock image").OnClick(func(*application.Context) { update("") })
	menu.AddSeparator()
	menu.Add("Quit TalkingDuck").OnClick(func(*application.Context) { app.Quit() })
	tray.SetMenu(menu)
	app.Event.On("mock:ready", func(*application.CustomEvent) { update("") })
	app.Event.On("mock:compact", func(*application.CustomEvent) { update("compact") })
	app.Event.On("mock:toggle", func(*application.CustomEvent) { update("styles") })
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) { update("") })
	return app.Run()
}
