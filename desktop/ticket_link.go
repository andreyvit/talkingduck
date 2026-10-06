//go:build darwin && cgo

package main

import (
	"fmt"
	"talkingduck/actions"
	"talkingduck/table"
)

func (v *inventoryView) ticketKnown(ticket string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, row := range v.state.Model.Rows {
		for _, item := range row.Matches("o", "t") {
			if item.Link.Target == ticket {
				return true
			}
		}
	}
	return false
}
func (v *inventoryView) opener() actions.Opener {
	if v.settings == nil {
		return actions.Opener{}
	}
	return actions.Opener{SettingsPath: v.settings.Path}
}
func (v *inventoryView) openTicket(ticket string) error {
	if !v.ticketKnown(ticket) {
		return fmt.Errorf("ticket is no longer in the inventory; refresh and try again")
	}
	return v.opener().Open(table.Action{Kind: "ticket", Target: ticket})
}
func (v *inventoryView) copyTicketURL(ticket string) error {
	if !v.ticketKnown(ticket) {
		return fmt.Errorf("ticket is no longer in the inventory; refresh and try again")
	}
	url, err := v.opener().TicketURL(ticket)
	if err != nil {
		return err
	}
	return actions.Copy(url)
}
func (v *inventoryView) openProject(kind, path string) error {
	target, err := folderTarget(path, v.cache.Paths())
	if err != nil {
		return err
	}
	return v.opener().Open(table.Action{Kind: kind, Target: target})
}
