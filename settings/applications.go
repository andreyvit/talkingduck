package settings

import (
	"fmt"
	"talkingduck/launchers"
)

type Applications struct {
	Markdown  launchers.Selection        `json:"markdown"`
	Editor    launchers.Selection        `json:"editor"`
	GitClient launchers.Selection        `json:"git_client"`
	Tracker   launchers.TrackerSelection `json:"tracker"`
}

func (a Applications) Defaults() Applications {
	if a.Markdown.ID == "" {
		a.Markdown.ID = "default"
	}
	if a.Editor.ID == "" {
		a.Editor.ID = "zed"
	}
	if a.GitClient.ID == "" {
		a.GitClient.ID = "sublime-merge"
	}
	if a.Tracker.ID == "" {
		a.Tracker.ID = "linear"
	}
	return a
}
func (s *Store) SetApplication(kind string, selection launchers.Selection) (Data, error) {
	if kind != "editor" && kind != "git_client" && kind != "markdown" {
		return Data{}, fmt.Errorf("unknown application setting %q", kind)
	}
	apps := launchers.Editors
	if kind == "git_client" {
		apps = launchers.GitClients
	}
	if kind == "markdown" {
		apps = launchers.MarkdownViewers
	}
	if err := launchers.ValidateSelection(apps, selection); err != nil {
		return Data{}, err
	}
	return s.update(func(d *Data) error {
		if kind == "git_client" {
			d.Applications.GitClient = selection
		} else if kind == "markdown" {
			d.Applications.Markdown = selection
		} else {
			d.Applications.Editor = selection
		}
		return nil
	})
}
func (s *Store) SetTracker(selection launchers.TrackerSelection) (Data, error) {
	configured, err := selection.Configure()
	if err != nil {
		return Data{}, err
	}
	return s.update(func(d *Data) error { d.Applications.Tracker = configured; return nil })
}
