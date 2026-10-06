//go:build darwin && cgo

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRevealSettingsUsesConfiguredFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom settings.json")
	if _, err := settingsRevealCommand(path); err == nil {
		t.Fatal("missing file accepted")
	}
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd, err := settingsRevealCommand(path)
	if err != nil || cmd.Executable != "/usr/bin/open" || len(cmd.Args) != 2 || cmd.Args[0] != "-R" || cmd.Args[1] != path {
		t.Fatal(cmd, err)
	}
}
