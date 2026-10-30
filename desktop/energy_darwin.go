//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
void tdObserveEnergy(void *, void *, void *);
*/
import "C"

import (
	"context"
	"github.com/wailsapp/wails/v3/pkg/application"
	"log"
	"sync/atomic"
	"talkingduck/telemetry"
)

var energyMask atomic.Int32
var energyChanges = make(chan struct{}, 1)
var discoveryVisibility = make(chan struct{}, 1)

//export tdEnergyChanged
func tdEnergyChanged(mask C.int) {
	energyMask.Store(int32(mask))
	select {
	case energyChanges <- struct{}{}:
	default:
	}
}

func observeEnergy(ctx context.Context, app *application.App, window, settingsWindow *application.WebviewWindow, v *inventoryView, monitoring *telemetry.Manager, agents bool, refresh func()) {
	C.tdObserveEnergy(window.NativeWindow(), v.markdown.window.NativeWindow(), settingsWindow.NativeWindow())
	var cancel context.CancelFunc
	var done <-chan struct{}
	defer func() {
		if cancel != nil {
			cancel()
			<-done
		}
	}()
	previous := int32(-1)
	for {
		select {
		case <-ctx.Done():
			return
		case <-energyChanges:
			mask := energyMask.Load()
			awake := mask&1 != 0
			if previous&9 != mask&9 || previous == -1 {
				select {
				case discoveryVisibility <- struct{}{}:
				default:
				}
			}
			log.Printf("energy: awake=%t dashboard=%t markdown=%t", awake, mask&2 != 0, mask&4 != 0)
			v.markdown.mu.Lock()
			v.markdown.obscured = mask&4 == 0
			v.markdown.mu.Unlock()
			v.markdown.signalVisibility()
			app.Event.Emit("energy:visibility", map[string]bool{"dashboard": mask&2 != 0, "markdown": mask&4 != 0})
			if (previous&1 != mask&1) || previous == -1 {
				v.cheap.SetPaused(!awake)
				v.expensive.SetPaused(!awake)
				monitoring.SetSuspended(!awake)
				if cancel != nil {
					cancel()
					<-done
					cancel = nil
				}
				if awake {
					cancel, done = v.startBackground(ctx, agents)
					v.refresh()
				}
			}

			if awake {
				refresh()
			}
			previous = mask
		}
	}
}

// The owner cancels and joins these sources before starting a new display session.
func (v *inventoryView) startBackground(ctx context.Context, agents bool) (context.CancelFunc, <-chan struct{}) {
	active, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		changesDone := make(chan struct{})
		go func() { defer close(changesDone); v.watchChanges(active, agents) }()
		v.watchSettings(active, agents)
		<-changesDone
	}()
	return cancel, done
}
