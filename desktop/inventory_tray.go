//go:build darwin && cgo

package main

import (
	"encoding/json"
	"talkingduck/inventory"
	"talkingduck/menubarimage"
)

// Keep the first listed task per checkout, matching the original prototype.
// Never present failed or unscanned cache entries as current activity.
func inventoryCells(snapshot *inventory.Snapshot) []menubarimage.Cell {
	var data struct {
		Repositories []struct {
			Cache inventory.RefreshState `json:"cache"`
			Git   struct {
				Cleanliness string `json:"cleanliness"`
				Error       string `json:"error"`
			} `json:"git"`
			Specs []struct {
				Workflow struct {
					Stage       string `json:"stage"`
					Progress    string `json:"progress"`
					Interaction string `json:"interaction"`
				} `json:"workflow"`
			} `json:"specs"`
		} `json:"repositories"`
	}
	b, _ := json.Marshal(snapshot.Overview)
	_ = json.Unmarshal(b, &data)
	cells := make([]menubarimage.Cell, len(data.Repositories))
	for i, repo := range data.Repositories {
		c := menubarimage.Cell{Number: i + 1, Stage: "?", State: "unknown"}
		if repo.Cache.State == "ready" && repo.Git.Error == "" {
			if len(repo.Specs) == 0 {
				if repo.Git.Cleanliness == "CLEAN" {
					c.Stage = "-"
					c.State = "idle"
				} else if repo.Git.Cleanliness == "DIRTY" || repo.Git.Cleanliness == "MARKDN" {
					c.Stage = "d"
					c.State = "idle"
				}
			} else {
				w := repo.Specs[0].Workflow
				if stage, ok := map[string]string{"HL": "H", "LL": "L", "EX": "E", "FIN": "F", "SQ": "Q", "REBA": "R", "PUSH": "P"}[w.Stage]; ok {
					c.Stage = stage
				}
				switch {
				case w.Interaction == "!" || w.Progress == "✓":
					c.State = "attention"
				case w.Progress == "▶":
					c.State = "running"
				}
			}
		}
		cells[i] = c
	}
	return cells
}
