//go:build darwin && cgo

package main

import (
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"talkingduck/cli"
	"talkingduck/inventory"
)

type inventoryState struct {
	cli.TableView
	Loading   bool   `json:"loading"`
	Error     string `json:"error"`
	UpdatedAt string `json:"updatedAt"`
}

type inventoryView struct {
	mu      sync.Mutex
	app     *application.App
	options inventory.Options
	state   inventoryState
}

func (v *inventoryView) publish() {
	v.mu.Lock()
	state := v.state
	v.mu.Unlock()
	v.app.Event.Emit("inventory:snapshot", state)
}

// Refresh coalesces concurrent requests and never blocks the window event loop.
func (v *inventoryView) refresh() {
	v.mu.Lock()
	if v.state.Loading {
		v.mu.Unlock()
		return
	}
	v.state.Loading = true
	v.state.Error = ""
	v.mu.Unlock()
	v.publish()
	go func() {
		snapshot, err := inventory.Build(v.options)
		var table cli.TableView
		if err == nil {
			table, err = cli.BuildTable(snapshot.Overview)
		}
		v.mu.Lock()
		v.state.Loading = false
		if err != nil {
			v.state.Error = err.Error()
			log.Printf("inventory refresh failed: %v", err)
		} else {
			v.state.TableView = table
			v.state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			log.Printf("inventory: %d table rows", len(table.Rows))
		}
		v.mu.Unlock()
		v.publish()
	}()
}
