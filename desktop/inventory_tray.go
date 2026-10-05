//go:build darwin && cgo

package main

import (
	"encoding/json"
	"talkingduck/inventory"
	"talkingduck/menubarimage"
	"talkingduck/settings"
)

// Keep the first listed task per checkout, matching the original prototype.
// Never present failed or unscanned cache entries as current activity.
func inventoryCells(snapshot *inventory.Snapshot, folders ...[]settings.Folder) []menubarimage.Cell {
	var data struct {
		Repositories []struct {
			Path  string                 `json:"path"`
			Cache inventory.RefreshState `json:"cache"`
			Git   struct {
				Cleanliness string `json:"cleanliness"`
				Error       string `json:"error"`
			} `json:"git"`
			Specs []struct {
				PrimarySession struct {
					Engine string `json:"engine"`
				} `json:"primary_session"`
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
	shortcuts := map[string]string{}
	if len(folders) > 0 {
		for _, folder := range folders[0] {
			if folder.Visible {
				shortcuts[folder.Path] = folder.MenubarItemShortcut
			}
		}
	}
	cells := []menubarimage.Cell{}
	for i, repo := range data.Repositories {
		c := menubarimage.Cell{Number: i + 1, Stage: "?", State: "unknown"}
		if len(folders) > 0 {
			c.Shortcut = shortcuts[repo.Path]
			if c.Shortcut == "" {
				continue
			}
		}
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
				c.Agent = repo.Specs[0].PrimarySession.Engine
				if stage, ok := map[string]string{"HL": "H", "LL": "L", "EX": "E", "FIN": "F", "SQ": "Q", "REBA": "R", "PUSH": "P", "PUSHED": "P"}[w.Stage]; ok {
					c.Stage = stage
				}
				switch {
				case w.Stage == "PUSHED":
					c.State = "idle"
				case w.Interaction == "!" || w.Progress == "✓":
					c.State = "attention"
				case w.Progress == "▶":
					c.State = "running"
				}
			}
		}
		cells = append(cells, c)
	}
	return cells
}
