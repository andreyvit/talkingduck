//go:build darwin && cgo

package main

import (
	"os/exec"
	"path/filepath"
	"talkingduck/inventory"
	"talkingduck/settings"
	"testing"
)

func TestFolderSettingsApplyWithoutScan(t *testing.T) {
	c, err := inventory.New(inventory.Options{NoAgents: true})
	if err != nil {
		t.Fatal(err)
	}
	v := &inventoryView{cache: c, settings: &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}, emit: func(inventoryState) {}}
	folder := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", folder).CombinedOutput(); err != nil {
		t.Fatal(string(out), err)
	}
	if err = v.AddMonitoredFolder(folder); err != nil {
		t.Fatal(err)
	}
	if err = v.UpdateFolderMenubarItemShortcut(folder, "a"); err != nil {
		t.Fatal(err)
	}
	folders, err := v.ListFolders()
	if err != nil || len(folders) != 1 || folders[0].Status != "pending" {
		t.Fatal(folders, err)
	}
	cells := inventoryCells(c.Snapshot(), []settings.Folder{folders[0].Folder})
	if len(cells) != 1 || cells[0].Shortcut != "a" {
		t.Fatal(cells)
	}
	if err = v.RemoveMonitoredFolder(folder); err != nil {
		t.Fatal(err)
	}
	folders, err = v.ListFolders()
	if err != nil || folders[0].Visible || folders[0].Status != "hidden" || len(c.Paths()) != 0 {
		t.Fatal(folders, err)
	}
	if len(inventoryCells(c.Snapshot(), []settings.Folder{folders[0].Folder})) != 0 {
		t.Fatal("hidden folder in tray")
	}
}
