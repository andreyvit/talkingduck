package launchers

// Catalog provenance and platform limits: docs/launchers.md.
func editor(id, name, bundle, linux, windows string) App {
	a := App{ID: id, Name: name, Platforms: map[string]Launch{}}
	if bundle != "" {
		a.Platforms["darwin"] = Launch{BundleID: bundle}
	}
	if linux != "" {
		a.Platforms["linux"] = Launch{Executables: []string{linux}, Args: []string{"$PATH"}}
	}
	if windows != "" {
		a.Platforms["windows"] = Launch{Executables: []string{windows}, Args: []string{"$PATH"}}
	}
	return a
}

var Editors = []App{
	editor("zed", "Zed", "dev.zed.Zed", "zed", "zed.exe"),
	editor("vscode", "Visual Studio Code", "com.microsoft.VSCode", "code", "Code.exe"),
	editor("cursor", "Cursor", "com.todesktop.230313mzl4w4u92", "", ""),
	editor("windsurf", "Windsurf", "com.exafunction.windsurf", "", ""),
	editor("vscodium", "VSCodium", "com.vscodium", "codium", "VSCodium.exe"),
	editor("vscode-insiders", "Visual Studio Code Insiders", "com.microsoft.VSCodeInsiders", "code-insiders", "Code - Insiders.exe"),
	editor("sublime-text", "Sublime Text", "com.sublimetext.4", "subl", "subl.exe"),
	editor("bbedit", "BBEdit", "com.barebones.bbedit", "", ""),
	editor("nova", "Nova", "com.panic.Nova", "", ""),
	editor("textmate", "TextMate", "com.macromates.TextMate", "", ""),
	editor("macvim", "MacVim", "org.vim.MacVim", "", ""),
	editor("neovide", "Neovide", "com.neovide.neovide", "neovide", "neovide.exe"),
	editor("vimr", "VimR", "com.qvacua.VimR", "", ""),
	editor("idea", "IntelliJ IDEA", "com.jetbrains.intellij", "idea", "idea64.exe"),
	editor("goland", "GoLand", "com.jetbrains.goland", "goland", "goland64.exe"),
	editor("webstorm", "WebStorm", "com.jetbrains.WebStorm", "webstorm", "webstorm64.exe"),
	editor("pycharm", "PyCharm", "com.jetbrains.PyCharm", "pycharm", "pycharm64.exe"),
	editor("phpstorm", "PhpStorm", "com.jetbrains.PhpStorm", "phpstorm", "phpstorm64.exe"),
	editor("rubymine", "RubyMine", "com.jetbrains.RubyMine", "rubymine", "rubymine64.exe"),
	editor("clion", "CLion", "com.jetbrains.CLion", "clion", "clion64.exe"),
	editor("rider", "Rider", "com.jetbrains.rider", "rider", "rider64.exe"),
	editor("rustrover", "RustRover", "com.jetbrains.RustRover", "rustrover", "rustrover64.exe"),
	editor("android-studio", "Android Studio", "com.google.android.studio", "studio", "studio64.exe"),
	editor("xcode", "Xcode", "com.apple.dt.Xcode", "", ""),
	editor("pulsar", "Pulsar", "dev.pulsar-edit.pulsar", "pulsar", ""),
	editor("kate", "Kate", "", "kate", ""),
	editor("gedit", "Gedit", "", "gedit", ""),
}
var GitClients = []App{
	editor("sublime-merge", "Sublime Merge", "com.sublimemerge", "smerge", "smerge.exe"),
	editor("fork", "Fork", "com.DanPristupov.Fork", "", ""),
	editor("tower", "Tower", "com.fournova.Tower3", "", ""),
	editor("sourcetree", "Sourcetree", "com.torusknot.SourceTreeNotMAS", "", ""),
	{ID: "github-desktop", Name: "GitHub Desktop", Platforms: map[string]Launch{
		"darwin": {Executables: []string{"github", "/Applications/GitHub Desktop.app/Contents/Resources/app/static/darwin/github.sh", "~/Applications/GitHub Desktop.app/Contents/Resources/app/static/darwin/github.sh"}, Args: []string{"$PATH"}},
	}},
	{ID: "gitkraken", Name: "GitKraken", Platforms: map[string]Launch{
		"darwin":  {Executables: []string{"gitkraken", "/Applications/GitKraken.app/Contents/MacOS/GitKraken", "~/Applications/GitKraken.app/Contents/MacOS/GitKraken"}, Args: []string{"-p", "$PATH"}},
		"linux":   {Executables: []string{"gitkraken"}, Args: []string{"-p", "$PATH"}},
		"windows": {Executables: []string{"gitkraken.exe"}, Args: []string{"-p", "$PATH"}},
	}},
}
