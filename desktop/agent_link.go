//go:build darwin && cgo

package main

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"
)

func (v *inventoryView) openAgent(link string) error {
	v.mu.Lock()
	known := link != "" && slices.Contains(v.state.AgentLinks, link)
	v.mu.Unlock()
	if !known {
		return fmt.Errorf("chat is no longer in the inventory; refresh and try again")
	}
	output, err := exec.Command("/usr/bin/open", link).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot open chat: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return nil
}
