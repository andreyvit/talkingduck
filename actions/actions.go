// Package actions executes typed table actions without shell interpolation.
package actions

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"talkingduck/launchers"
	"talkingduck/settings"
	"talkingduck/table"
	"time"
)

func Open(a table.Action) error {
	return (Opener{}).Open(a)
}

type Opener struct{ SettingsPath string }

func (o Opener) preferences() (settings.Applications, error) {
	path := o.SettingsPath
	if path == "" {
		path = settings.DefaultPath()
	}
	d, err := (&settings.Store{Path: path}).Load()
	return d.Applications.Defaults(), err
}
func (o Opener) TicketURL(id string) (string, error) {
	p, err := o.preferences()
	if err != nil {
		return "", err
	}
	return p.Tracker.TicketLink(id)
}
func (o Opener) Open(a table.Action) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	target := a.Target
	if target == "" {
		return fmt.Errorf("no open target")
	}
	if a.Kind == "agent" && strings.HasPrefix(target, "talkingduck-grok:") {
		return OpenGrok(strings.TrimPrefix(target, "talkingduck-grok:"))
	}
	switch a.Kind {
	case "new-codex", "new-claude", "new-grok":
		return launchers.OpenAgent(strings.TrimPrefix(a.Kind, "new-"), target)
	case "reveal-spec":
		return launchers.Reveal(target)
	case "spec":
		p, err := o.preferences()
		if err != nil {
			return err
		}
		return launchers.OpenMarkdown(p.Markdown, p.Editor, target)
	case "ticket":
		var err error
		target, err = o.TicketURL(target)
		if err != nil {
			return err
		}
		return launchers.OpenURL(ctx, target)
	case "editor", "git":
		p, err := o.preferences()
		if err != nil {
			return err
		}
		if a.Kind == "editor" {
			return launchers.OpenProject(launchers.Editors, p.Editor, target)
		}
		return launchers.OpenProject(launchers.GitClients, p.GitClient, target)
	case "agent", "folder":
	default:
		return fmt.Errorf("%s opening is not available yet", a.Kind)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		args := []string{}
		if a.Kind == "folder" {
			args = append(args, "-a", "Finder")
		}
		args = append(args, target)
		cmd = exec.CommandContext(ctx, "/usr/bin/open", args...)
	} else {
		cmd = exec.CommandContext(ctx, "xdg-open", target)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("open failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}
func Copy(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command, args := "pbcopy", []string{}
	if runtime.GOOS != "darwin" {
		if _, err := exec.LookPath("wl-copy"); err == nil {
			command = "wl-copy"
		} else {
			command, args = "xclip", []string{"-selection", "clipboard"}
		}
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}
