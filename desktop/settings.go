//go:build darwin && cgo

package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"talkingduck/inventory"
	"talkingduck/settings"
	"talkingduck/worktree"
	"time"
)

func (v *inventoryView) ListFolders() ([]settings.FolderStatus, error) {
	data, err := v.settings.Load()
	if err != nil {
		return nil, err
	}
	snapshot := v.cache.Snapshot()
	states := map[string]*inventory.RefreshState{}
	for _, repo := range snapshot.Details.Repositories {
		states[repo.Path] = repo.Cache
	}
	result := []settings.FolderStatus{}
	for _, folder := range data.Folders {
		if !worktree.IsGit(folder.Path) {
			continue
		}
		row := settings.FolderStatus{Folder: folder, Status: "hidden"}
		if folder.Visible {
			row.Status = "pending"
			if c := states[folder.Path]; c != nil {
				row.Status, row.Error = c.State, c.Error
			}
		}
		if st, e := os.Stat(folder.Path); e != nil || !st.IsDir() {
			row.Status = "missing"
		}
		result = append(result, row)
	}
	return result, nil
}
func (v *inventoryView) applyFolders(data settings.Data) error {
	cache, ok := v.cache.(interface{ SetPaths([]string) error })
	if !ok {
		return fmt.Errorf("folder configuration unavailable")
	}
	if !v.folderOverride && !slices.Equal(v.cache.Paths(), data.Paths()) {
		if err := cache.SetPaths(data.Paths()); err != nil {
			return err
		}
		select {
		case v.foldersChanged <- struct{}{}:
		default:
		}
	}
	v.render("", true)
	if v.settingsChanged != nil {
		v.settingsChanged()
	}
	return nil
}
func (v *inventoryView) AddMonitoredFolder(path string) error {
	v.folderMu.Lock()
	defer v.folderMu.Unlock()
	data, err := v.settings.SetVisible(path, true)
	if err != nil {
		return err
	}
	v.folderOverride = false
	return v.applyFolders(data)
}
func (v *inventoryView) RemoveMonitoredFolder(path string) error {
	v.folderMu.Lock()
	defer v.folderMu.Unlock()
	data, err := v.settings.SetVisible(path, false)
	if err != nil {
		return err
	}
	v.folderOverride = false
	return v.applyFolders(data)
}
func (v *inventoryView) UpdateFolderMenubarItemShortcut(path, shortcut string) error {
	v.folderMu.Lock()
	defer v.folderMu.Unlock()
	data, err := v.settings.SetShortcut(path, shortcut)
	if err != nil {
		return err
	}
	v.folderOverride = false
	return v.applyFolders(data)
}
func (v *inventoryView) watchSettings(ctx context.Context, agents bool) {
	watchDone := make(chan struct{})
	defer func() { <-watchDone }()
	go func() {
		defer close(watchDone)
		v.settings.Watch(ctx, func(settings.Data) {
			v.folderMu.Lock()
			defer v.folderMu.Unlock()
			// Re-read under the mutation lock; an older watch callback cannot undo an edit.
			data, err := v.settings.Load()
			if err == nil {
				err = v.applyFolders(data)
			}
			if err != nil {
				v.settingsError(err)
			}
		}, v.settingsError)
	}()
	if !agents {
		<-ctx.Done()
		return
	}
	// Discovery only supplies the Settings folder picker. It need not scan every
	// provider's session metadata while that window is hidden.
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		if ctx.Err() != nil {
			return
		}
		select {
		case <-discoveryVisibility:
		default:
		}
		var tick <-chan time.Time
		if energyMask.Load()&8 != 0 {
			paths := inventory.DiscoverFolders("")
			v.folderMu.Lock()
			_, err := v.settings.Discover(paths)
			v.folderMu.Unlock()
			if err != nil {
				v.settingsError(err)
			} else if v.settingsChanged != nil {
				v.settingsChanged()
			}
			timer = time.NewTimer(time.Minute)
			tick = timer.C
		}
		select {
		case <-ctx.Done():
			return
		case <-discoveryVisibility:
		case <-tick:
		}
		if timer != nil {
			timer.Stop()
			timer = nil
		}
	}
}
