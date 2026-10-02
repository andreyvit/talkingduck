//go:build darwin && cgo

package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Workspace selection will move to preferences; see docs/future-settings.md.
const linearWorkspace = "bubblehouse"

var ticketPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*-[0-9]+$`)

func ticketURL(ticket string) (string, error) {
	if !ticketPattern.MatchString(ticket) {
		return "", fmt.Errorf("invalid ticket reference: %q", ticket)
	}
	return "https://linear.app/" + linearWorkspace + "/issue/" + ticket, nil
}

func (v *inventoryView) openTicket(ticket string) error {
	url, err := ticketURL(ticket)
	if err != nil {
		return err
	}
	v.mu.Lock()
	known := false
	for _, row := range v.state.Rows {
		if len(row) > 2 {
			for _, id := range strings.Split(row[2], "\n") {
				if id == ticket {
					known = true
				}
			}
		}
	}
	v.mu.Unlock()
	if !known {
		return fmt.Errorf("ticket is no longer in the inventory; refresh and try again")
	}
	output, err := exec.Command("/usr/bin/open", url).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot open Linear ticket: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return nil
}
