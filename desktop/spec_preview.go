//go:build darwin && cgo

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func (v *inventoryView) openSpec(path string) error {
	v.mu.Lock()
	known := slices.Contains(v.state.SpecPaths, path)
	v.mu.Unlock()
	if path == "" || !filepath.IsAbs(path) || !known {
		return fmt.Errorf("spec is no longer in the inventory; refresh and try again")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot open spec: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("spec is not a regular file: %s", path)
	}
	// Pass the filename directly, with no shell interpretation. Keep the preview
	// app choice here until application preferences are introduced.
	output, err := exec.Command("/usr/bin/open", "-a", "Marked", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot open spec in Marked: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return nil
}
