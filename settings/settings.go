// Package settings owns the shared, user-configured folder list.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Folder struct {
	Path                string `json:"path"`
	Visible             bool   `json:"visible"`
	MenubarItemShortcut string `json:"menubar_item_shortcut"`
	Discovered          bool   `json:"discovered,omitempty"`
}
type Data struct {
	Version int      `json:"version"`
	Folders []Folder `json:"folders"`
}
type Store struct {
	Path string
	mu   sync.Mutex
}

func DefaultPath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "zoo", "talkingduck-settings.json")
}
func NormalizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("folder path is required")
	}
	if path == "~" {
		home, e := os.UserHomeDir()
		return home, e
	}
	if strings.HasPrefix(path, "~/") {
		home, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	path, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	if real, e := filepath.EvalSymlinks(path); e == nil {
		path = real
	}
	return path, nil
}
func ValidShortcut(s string) bool {
	return s == "" || len(s) == 2 && s[0] == '1' && s[1] >= '0' && s[1] <= '9' || len(s) == 1 && strings.Contains("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789", s)
}
func (d Data) Paths() []string {
	r := []string{}
	for _, f := range d.Folders {
		if f.Visible {
			r = append(r, f.Path)
		}
	}
	return r
}
func (s *Store) Load() (Data, error) {
	b, e := os.ReadFile(s.Path)
	if os.IsNotExist(e) {
		return Data{Version: 1, Folders: []Folder{}}, nil
	}
	if e != nil {
		return Data{}, e
	}
	var d Data
	if e = json.Unmarshal(b, &d); e != nil {
		return d, fmt.Errorf("invalid settings: %w", e)
	}
	if d.Version != 1 {
		return d, fmt.Errorf("unsupported settings version %d", d.Version)
	}
	seen := map[string]bool{}
	for i, f := range d.Folders {
		p, e := NormalizePath(f.Path)
		if e != nil {
			return d, e
		}
		if seen[p] {
			return d, fmt.Errorf("duplicate folder: %s", p)
		}
		seen[p] = true
		d.Folders[i].Path = p
		if !ValidShortcut(f.MenubarItemShortcut) {
			return d, fmt.Errorf("invalid menu bar shortcut for %s", p)
		}
	}
	return d, nil
}
func (s *Store) update(fn func(*Data) error) (Data, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := os.MkdirAll(filepath.Dir(s.Path), 0700); e != nil {
		return Data{}, e
	}
	lock, e := os.OpenFile(s.Path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return Data{}, e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); e != nil {
		return Data{}, e
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	d, e := s.Load()
	if e != nil {
		return d, e
	}
	before, _ := json.Marshal(d)
	if e = fn(&d); e != nil {
		return d, e
	}
	after, _ := json.Marshal(d)
	if string(before) == string(after) {
		return d, nil
	}
	b, e := json.MarshalIndent(d, "", "  ")
	if e != nil {
		return d, e
	}
	f, e := os.CreateTemp(filepath.Dir(s.Path), ".talkingduck-settings-*")
	if e != nil {
		return d, e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(append(b, '\n')); e == nil {
		e = f.Sync()
	}
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e == nil {
		e = os.Rename(f.Name(), s.Path)
	}
	return d, e
}

// SetVisible retains hidden entries, so rediscovery never re-enables them.
func (s *Store) SetVisible(path string, visible bool) (Data, error) {
	path, e := NormalizePath(path)
	if e != nil {
		return Data{}, e
	}
	if visible {
		st, e := os.Stat(path)
		if e != nil {
			return Data{}, e
		}
		if !st.IsDir() {
			return Data{}, fmt.Errorf("not a folder: %s", path)
		}
	}
	return s.update(func(d *Data) error {
		for i := range d.Folders {
			if d.Folders[i].Path == path {
				d.Folders[i].Visible = visible
				return nil
			}
		}
		if !visible {
			return fmt.Errorf("folder is not configured: %s", path)
		}
		d.Folders = append(d.Folders, Folder{Path: path, Visible: true})
		return nil
	})
}
func (s *Store) SetShortcut(path, shortcut string) (Data, error) {
	path, e := NormalizePath(path)
	if e != nil {
		return Data{}, e
	}
	if !ValidShortcut(shortcut) {
		return Data{}, fmt.Errorf("shortcut must be empty, one letter, or a number from 0 to 19")
	}
	return s.update(func(d *Data) error {
		for i := range d.Folders {
			if d.Folders[i].Path == path {
				d.Folders[i].MenubarItemShortcut = shortcut
				return nil
			}
		}
		return fmt.Errorf("folder is not configured: %s", path)
	})
}
func (s *Store) Discover(paths []string) (Data, error) {
	return s.update(func(d *Data) error {
		seen := map[string]bool{}
		for _, f := range d.Folders {
			seen[f.Path] = true
		}
		for _, p := range paths {
			p, e := NormalizePath(p)
			if e != nil || seen[p] {
				continue
			}
			seen[p] = true
			d.Folders = append(d.Folders, Folder{Path: p, Discovered: true})
		}
		return nil
	})
}

// Watch keeps other running clients in sync with atomic settings replacements.
// This tiny file check is separate from all inventory work.
func (s *Store) Watch(ctx context.Context, changed func(Data), failed func(error)) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	previous := ""
	initialized := false
	for {
		b, err := os.ReadFile(s.Path)
		if os.IsNotExist(err) {
			b = nil
			err = nil
		}
		if err != nil {
			failed(err)
		} else if !initialized || string(b) != previous {
			d, e := s.Load()
			if e != nil {
				failed(e)
			} else {
				previous = string(b)
				initialized = true
				changed(d)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// FolderStatus adds runtime status without persisting it as configuration.
type FolderStatus struct {
	Folder
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
