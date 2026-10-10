//go:build darwin && cgo

package main

import (
	"fmt"
	"slices"
	"talkingduck/actions"
	"talkingduck/table"
)

func (v *inventoryView) openAgent(link string) error {
	v.mu.Lock()
	known := link != "" && slices.Contains(v.state.AgentLinks, link)
	for _, row := range v.state.Model.Rows {
		for _, item := range row.Actions {
			if item.Link != nil && item.Link.Kind == "agent" && item.Link.Target == link && link != "" {
				known = true
			}
		}
	}
	v.mu.Unlock()
	if !known {
		return fmt.Errorf("chat is no longer in the inventory; refresh and try again")
	}
	return actions.Open(table.Action{Kind: "agent", Target: link})
}
