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

func folderTarget(path string, known []string) (string, error) {
	if !filepath.IsAbs(path) || !slices.Contains(known, path) {
		return "", fmt.Errorf("folder is not in the inventory")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("cannot open folder: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a folder: %s", path)
	}
	return path, nil
}
func (v *inventoryView) openFolder(path string) error {
	target, err := folderTarget(path, v.cache.Paths())
	if err != nil {
		return err
	}
	output, err := exec.Command("/usr/bin/open", "-a", "Finder", target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cannot open folder in Finder: %s (%w)", strings.TrimSpace(string(output)), err)
	}
	return nil
}
