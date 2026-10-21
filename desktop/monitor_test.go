//go:build darwin && cgo

package main

import (
	"talkingduck/settings"
	"talkingduck/telemetry"
	"testing"
	"time"
)

func TestMonitorArtVisibilityAndOutage(t *testing.T) {
	now := time.Now()
	failed := now.Add(-3 * time.Second)
	zero := 0.0
	s := telemetry.Snapshot{Ping: telemetry.Ping{State: "unreachable", FailedSince: &failed}, Providers: []telemetry.Provider{{ID: "codex", RecentlyUsed: true, RemainingPercent: &zero}, {ID: "claude"}, {ID: "grok", RecentlyUsed: true}}}
	art := monitorArt(s, settings.Menubar{Grok: "off", Claude: "on"}, now)
	if art.Ping != "3 s" || !art.PingFailed || len(art.Bars) != 2 || art.Bars[0].Remaining == nil || *art.Bars[0].Remaining != 0 {
		t.Fatal(art)
	}
	art = monitorArt(s, settings.Menubar{Ping: "off", Codex: "off", Claude: "off", Grok: "off"}, now)
	if art.Ping != "" || len(art.Bars) != 0 {
		t.Fatal(art)
	}
}
