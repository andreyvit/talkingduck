package launchers

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

var MarkdownViewers = []App{
	{ID: "default", Name: "(Default)", Platforms: map[string]Launch{"darwin": {}, "linux": {}, "windows": {}}},
	{ID: "text-editor", Name: "(Text editor)", Platforms: map[string]Launch{"darwin": {}, "linux": {}, "windows": {}}},
	editor("markdownreader", "MarkDownReader", "com.markdownreader.MarkDownReader", "", ""),
	editor("marked", "Marked", "com.brettterpstra.marked.pro", "", ""),
	editor("macmd-viewer", "MacMD Viewer", "com.arthur.MarkdownViewer", "", ""),
	editor("macdown", "MacDown", "com.uranusjr.macdown", "", ""),
	editor("typora", "Typora", "abnerworks.Typora", "typora", "Typora.exe"),
	editor("marktext", "MarkText", "com.github.marktext.marktext", "marktext", "MarkText.exe"),
	editor("ghostwriter", "ghostwriter", "", "ghostwriter", "ghostwriter.exe"),
}

func OpenMarkdown(viewer, editor Selection, path string) error {
	switch viewer.ID {
	case "", "default":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return OpenURL(ctx, path)
	case "text-editor":
		return OpenProject(Editors, editor, path)
	default:
		return OpenProject(MarkdownViewers, viewer, path)
	}
}

// Reveal selects the file where supported; Linux opens its containing directory.
func Reveal(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "/usr/bin/open", "-R", path).Run()
	case "windows":
		return (Command{Executable: "explorer.exe", Args: []string{"/select,", path}}).Start()
	default:
		return OpenURL(ctx, filepath.Dir(path))
	}
}
