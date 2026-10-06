//go:build darwin && cgo

package main

import (
	"talkingduck/cli"
	"talkingduck/table"
	"testing"
)

func TestTicketMustBelongToCurrentInventory(t *testing.T) {
	v := &inventoryView{state: inventoryState{TableView: cli.TableView{Model: table.Table{Rows: []table.Row{{Cells: []table.Cell{{Lines: []table.Line{{Items: []table.Item{{Chord: "t", Link: &table.Action{Kind: "ticket", Target: "DEV-1"}}}}}}}}}}}}}
	if !v.ticketKnown("DEV-1") || v.ticketKnown("DEV-2") {
		t.Fatal("ticket membership mismatch")
	}
	if err := v.openTicket("DEV-2"); err == nil {
		t.Fatal("opened stale ticket")
	}
}
