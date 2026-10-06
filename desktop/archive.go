//go:build darwin && cgo

package main

import (
	"fmt"
	"talkingduck/cli"
)

func (v *inventoryView) archiveRow(folder, spec string) error {
	cache, ok := v.cache.(interface{ ArchiveRow(string, string) error })
	if !ok {
		return fmt.Errorf("archiving is unavailable")
	}
	if err := cache.ArchiveRow(folder, spec); err != nil {
		return err
	}
	if v.settingsChanged != nil {
		v.settingsChanged()
	}
	snapshot := v.cache.Snapshot()
	table, err := cli.BuildTable(snapshot.Overview)
	v.publishTable(snapshot, table, err)
	return err
}
