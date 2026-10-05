package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Mode is shared by both binaries. Explicit modes are mutually exclusive.
// Legacy one-shot flags also retain one-shot behavior when no mode was supplied.
func (o *Options) Mode(defaultMode string) (string, error) {
	modes := []string{}
	for _, m := range []struct {
		on   bool
		name string
	}{{o.AI, "ai"}, {o.Status, "status"}, {o.TUI, "tui"}, {o.GUI, "gui"}} {
		if m.on {
			modes = append(modes, m.name)
		}
	}
	if len(modes) > 1 {
		return "", fmt.Errorf("choose only one of -ai, -status, -tui, -gui")
	}
	if len(modes) == 1 {
		return modes[0], nil
	}
	if o.Details || o.AgentsOnly {
		return "status", nil
	}
	return defaultMode, nil
}

// LaunchGUI opens the installed app, forwarding this invocation's options.
func LaunchGUI(args []string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("the GUI requires macOS")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	app := filepath.Join(home, "Applications", "TalkingDuck.app")
	if _, err = os.Stat(app); err != nil {
		return fmt.Errorf("GUI not installed; run ./desktop/build.sh: %w", err)
	}
	command := []string{"-n", app, "--args", "-gui"}
	for _, arg := range args {
		if arg != "-gui" && arg != "--gui" && arg != "-gui=true" && arg != "--gui=true" {
			command = append(command, arg)
		}
	}
	return exec.Command("/usr/bin/open", command...).Run()
}
