package launchers

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Quick detection intentionally avoids Spotlight, recursive scans, or launching
// applications. Custom handles false negatives (portable/custom installs).
func macAppPath(a App) string {
	home, _ := os.UserHomeDir()
	names := []string{a.Name}
	switch a.ID {
	case "marked":
		names = append(names, "Marked 2")
	case "vscode-insiders":
		names = append(names, "Visual Studio Code - Insiders")
	case "idea":
		names = append(names, "IntelliJ IDEA Ultimate", "IntelliJ IDEA CE")
	}
	for _, root := range []string{"/Applications", filepath.Join(home, "Applications"), "/Applications/Setapp"} {
		for _, name := range names {
			path := filepath.Join(root, name+".app")
			if info, err := os.Stat(filepath.Join(path, "Contents", "Info.plist")); err == nil && info.Mode().IsRegular() {
				return path
			}
		}
	}
	return ""
}
func DetectedApps(apps []App) []App {
	result := Available(apps, runtime.GOOS)
	for i := range result {
		a := &result[i]
		if a.ID == "built-in" || a.ID == "custom" || a.ID == "default" || a.ID == "text-editor" {
			continue
		}
		found := false
		if runtime.GOOS == "darwin" {
			found = macAppPath(*a) != ""
		} else {
			for _, name := range a.Platforms[runtime.GOOS].Executables {
				if _, err := lookup(name); err == nil {
					found = true
					break
				}
			}
		}
		a.Installed = &found
	}
	return result
}
func appLess(a, b App) bool { return strings.ToLower(a.Name) < strings.ToLower(b.Name) }
