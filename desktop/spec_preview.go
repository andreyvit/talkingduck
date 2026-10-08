//go:build darwin && cgo

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"talkingduck/actions"
	"talkingduck/table"
)

func (v *inventoryView) openSpec(path string) error {
	return v.specAction(path, "spec")
}
func (v *inventoryView) specAction(path, kind string) error {
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
	return (actions.Opener{SettingsPath: v.settings.Path}).Open(table.Action{Kind: kind, Target: path})
}
