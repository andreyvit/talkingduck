//go:build darwin && cgo

package main

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"talkingduck/menubarimage"
	"talkingduck/settings"
	"talkingduck/telemetry"
)

func monitorConfig(m settings.Menubar) telemetry.Config {
	m = m.Defaults()
	return telemetry.Config{PingTarget: m.PingTarget, PingEnabled: m.Ping != "off", Codex: m.Codex, Claude: m.Claude, Grok: m.Grok}
}
func monitorArt(s telemetry.Snapshot, m settings.Menubar, now time.Time) menubarimage.Monitor {
	m = m.Defaults()
	art := menubarimage.Monitor{}
	if m.Ping != "off" {
		switch s.Ping.State {
		case "reachable":
			if s.Ping.RTTMS != nil {
				art.PingMS = *s.Ping.RTTMS
				art.PingDigits = s.Ping.Digits
				art.Ping = fmt.Sprintf("%.0f", math.Min(9999, *s.Ping.RTTMS))
			} else {
				art.Ping = "—"
			}
		case "unreachable":
			art.PingFailed = true
			if s.Ping.FailedSince != nil {
				seconds := max(0, int(now.Sub(*s.Ping.FailedSince).Seconds()))
				if seconds > 9999 {
					art.Ping = "9999+s"
				} else {
					art.Ping = fmt.Sprintf("%d s", seconds)
				}
			} else {
				art.Ping = "—"
			}
		case "error":
			art.Ping = "!"
			art.PingFailed = true
		default:
			art.Ping = "…"
		}
	}
	modes := map[string]string{"codex": m.Codex, "claude": m.Claude, "grok": m.Grok}
	for _, p := range s.Providers {
		mode := modes[p.ID]
		if mode == "off" || mode != "on" && !p.RecentlyUsed {
			continue
		}
		art.Bars = append(art.Bars, menubarimage.LimitBar{ID: p.ID, Remaining: p.RemainingPercent, Stale: p.State == "stale"})
	}
	return art
}
func registerMenubarSettings(app *application.App, store *settings.Store, changed func()) {
	var mu sync.Mutex
	latest := map[string]int{}
	app.Event.On("settings:menubar", func(e *application.CustomEvent) {
		var request struct {
			Client   string           `json:"client"`
			Revision int              `json:"revision"`
			Value    settings.Menubar `json:"value"`
		}
		b, err := json.Marshal(e.Data)
		if err != nil || json.Unmarshal(b, &request) != nil {
			return
		}
		go func() {
			mu.Lock()
			defer mu.Unlock()
			if request.Revision < latest[request.Client] {
				return
			}
			latest[request.Client] = request.Revision
			_, err := store.SetMenubar(request.Value)
			message := ""
			if err != nil {
				message = err.Error()
			} else {
				changed()
			}
			app.Event.Emit("settings:menubar-result", map[string]any{"client": request.Client, "revision": request.Revision, "error": message})
		}()
	})
}
