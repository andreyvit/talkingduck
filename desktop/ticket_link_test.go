//go:build darwin && cgo

package main

import "testing"

func TestTicketURL(t *testing.T) {
	for _, ticket := range []string{"DEV-123", "BUB-004"} {
		url, err := ticketURL(ticket)
		if err != nil || url != "https://linear.app/bubblehouse/issue/"+ticket {
			t.Fatalf("%s: %s %v", ticket, url, err)
		}
	}
	for _, ticket := range []string{"--", "DEV-1\nBUB-2", "https://example.com", "DEV-1?x=y", "../DEV-1"} {
		if _, err := ticketURL(ticket); err == nil {
			t.Fatalf("accepted %q", ticket)
		}
	}
}
