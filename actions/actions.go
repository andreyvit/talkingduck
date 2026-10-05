// Package actions executes typed table actions without shell interpolation.
package actions

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"talkingduck/table"
	"time"
)

func Open(a table.Action) error {
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
	case "ticket":
		target = "https://linear.app/bubblehouse/issue/" + target
	case "agent", "spec", "folder":
	default:
		return fmt.Errorf("%s opening is not available yet", a.Kind)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		args := []string{}
		if a.Kind == "spec" {
			args = append(args, "-a", "Marked")
		}
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
