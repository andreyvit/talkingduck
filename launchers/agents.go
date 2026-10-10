package launchers

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// AgentURL opens a fresh desktop composer, not an existing session or a sent prompt.
func AgentURL(engine, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("agent project path must be absolute")
	}
	switch engine {
	case "codex":
		return "codex://threads/new?" + url.Values{"path": {path}}.Encode(), nil
	case "claude":
		return "claude://code/new?" + url.Values{"folder": {path}}.Encode(), nil
	default:
		return "", fmt.Errorf("unknown desktop agent %q", engine)
	}
}

// GrokCommand creates a fresh terminal. cmux's --command is shell text, so quote
// both arguments; never interpolate a project path as shell code.
func GrokCommand(path, executable, cmux string) Command {
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
	return Command{cmux, []string{"new-workspace", "--name", "Grok · " + filepath.Base(path),
		"--cwd", path, "--focus", "true", "--command", quote(executable) + " --cwd " + quote(path)}}
}

func OpenAgent(engine, path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("agent project path must be absolute")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("agent project: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("agent project is not a folder")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if engine != "grok" {
		target, err := AgentURL(engine, path)
		if err != nil {
			return err
		}
		return OpenURL(ctx, target)
	}
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("new Grok sessions currently require cmux on macOS")
	}
	grok, err := lookup("grok")
	if err != nil {
		grok, err = lookup(expandHome("~/.grok/bin/grok"))
	}
	if err != nil {
		return fmt.Errorf("Grok Build launcher not found")
	}
	cmux, err := lookup("cmux")
	if err != nil {
		cmux, err = lookup("/Applications/cmux.app/Contents/Resources/bin/cmux")
	}
	if err != nil {
		return fmt.Errorf("cmux launcher not found")
	}
	cmd := GrokCommand(path, grok, cmux)
	if err := runAgentCommand(ctx, cmd); err != nil {
		return err
	}
	return runAgentCommand(ctx, Command{"/usr/bin/open", []string{"-a", "cmux"}})
}

func runAgentCommand(ctx context.Context, command Command) error {
	out, err := exec.CommandContext(ctx, command.Executable, command.Args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("agent launch failed: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}
