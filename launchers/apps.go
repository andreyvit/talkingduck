// Package launchers opens projects and tickets without depending on a UI or settings store.
package launchers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Command struct {
	Executable string
	Args       []string
}
type Launch struct {
	BundleID    string
	Executables []string
	Args        []string
}
type App struct {
	Installed *bool             `json:"installed,omitempty"`
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Platforms map[string]Launch `json:"-"`
}
type Selection struct {
	ID   string `json:"id"`
	Path string `json:"path,omitempty"`
	Args string `json:"args,omitempty"`
}

func Available(apps []App, platform string) []App {
	result := []App{}
	for _, a := range apps {
		if _, ok := a.Platforms[platform]; ok {
			result = append(result, a)
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return appLess(result[i], result[j]) })
	return append(result, App{ID: "custom", Name: "Custom"})
}
func ValidateSelection(apps []App, s Selection) error {
	if s.ID == "custom" {
		if strings.TrimSpace(s.Path) == "" {
			return fmt.Errorf("custom executable or app path is required")
		}
		args, err := ParseArgs(s.Args)
		if err != nil {
			return err
		}
		for _, arg := range args {
			if _, err = expand(arg, map[string]string{"PATH": "project"}); err != nil {
				return err
			}
		}
		return nil
	}
	for _, a := range apps {
		if a.ID == s.ID {
			return nil
		}
	}
	return fmt.Errorf("unknown application %q", s.ID)
}

// Plan produces argv without invoking a shell. lookup may be replaced in tests.
func Plan(apps []App, s Selection, platform, path string, lookup func(string) (string, error)) (Command, error) {
	if err := ValidateSelection(apps, s); err != nil {
		return Command{}, err
	}
	if path == "" {
		return Command{}, fmt.Errorf("project path is required")
	}
	if s.ID == "custom" {
		args, err := ParseArgs(s.Args)
		if err != nil {
			return Command{}, err
		}
		for i, arg := range args {
			args[i], err = expand(arg, map[string]string{"PATH": path})
			if err != nil {
				return Command{}, err
			}
		}
		executable := expandHome(s.Path)
		if platform == "darwin" && strings.HasSuffix(strings.ToLower(executable), ".app") {
			// A new instance ensures custom arguments are delivered even when the app is open.
			return Command{"/usr/bin/open", append([]string{"-n", "-a", executable, "--args"}, args...)}, nil
		}
		return Command{executable, args}, nil
	}
	for _, a := range apps {
		if a.ID != s.ID {
			continue
		}
		spec, ok := a.Platforms[platform]
		if !ok {
			return Command{}, fmt.Errorf("%s is not supported on %s", a.Name, platform)
		}
		if spec.BundleID != "" {
			return Command{"/usr/bin/open", []string{"-b", spec.BundleID, path}}, nil
		}
		for _, name := range spec.Executables {
			if resolved, err := lookup(expandHome(name)); err == nil {
				args := append([]string(nil), spec.Args...)
				for i, arg := range args {
					args[i], _ = expand(arg, map[string]string{"PATH": path})
				}
				return Command{resolved, args}, nil
			}
		}
		return Command{}, fmt.Errorf("%s launcher not found; install its command-line launcher or choose Custom", a.Name)
	}
	return Command{}, fmt.Errorf("unknown application %q", s.ID)
}
func expandHome(s string) string {
	if strings.HasPrefix(s, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, s[2:])
	}
	return s
}
func lookup(name string) (string, error) {
	if p, e := exec.LookPath(name); e == nil {
		return p, nil
	}
	// GUI apps often inherit a minimal PATH.
	for _, base := range []string{"/opt/homebrew/bin", "/usr/local/bin", expandHome("~/.local/bin")} {
		if !strings.ContainsAny(name, "/\\") {
			if p, e := exec.LookPath(filepath.Join(base, name)); e == nil {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("executable not found: %s", name)
}
func OpenProject(apps []App, s Selection, path string) error {
	if path == "" {
		return fmt.Errorf("project path is required")
	}
	if runtime.GOOS == "darwin" && s.ID != "custom" {
		for _, app := range apps {
			if app.ID == s.ID && app.Platforms["darwin"].BundleID != "" {
				if installed := macAppPath(app); installed != "" {
					return (Command{Executable: "/usr/bin/open", Args: []string{"-a", installed, path}}).Start()
				}
			}
		}
	}
	cmd, err := Plan(apps, s, runtime.GOOS, path, lookup)
	if err != nil {
		return err
	}
	return cmd.Start()
}

// Start detaches GUI lifetime from the caller; Wait reaps the child without blocking the UI.
func (c Command) Start() error {
	if c.Executable == "/usr/bin/open" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, c.Executable, c.Args...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("open application: %s (%w)", strings.TrimSpace(string(out)), err)
		}
		return nil
	}
	cmd := exec.Command(c.Executable, c.Args...)
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch %s: %w", c.Executable, err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// OpenURL waits only for the OS URL dispatcher, not the application lifetime.
func OpenURL(ctx context.Context, target string) error {
	command, args := "xdg-open", []string{target}
	switch runtime.GOOS {
	case "darwin":
		command = "/usr/bin/open"
	case "windows":
		command = "rundll32.exe"
		args = []string{"url.dll,FileProtocolHandler", target}
	}
	out, err := exec.CommandContext(ctx, command, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("open: %s (%w)", strings.TrimSpace(string(out)), err)
	}
	return nil
}
