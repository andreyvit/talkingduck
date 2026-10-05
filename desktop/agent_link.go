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
	v.mu.Unlock()
	if !known {
		return fmt.Errorf("chat is no longer in the inventory; refresh and try again")
	}
	return actions.Open(table.Action{Kind: "agent", Target: link})
}
