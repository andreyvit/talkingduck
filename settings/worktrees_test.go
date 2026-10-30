package settings

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDiscoverRegisteredWorktreesAndShortcuts(t *testing.T) {
	repo := gitFolder(t)
	git := func(args ...string) {
		t.Helper()
		args = append([]string{"-C", repo}, args...)
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	git("-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Initial")
	children := map[string]string{}
	for _, tc := range []struct{ name, env string }{
		{"explicit", "export AGENT_INDEX=11\n"},
		{"back2", "AGENT_INDEX=11\n"}, {"back19", ""}, {"implicit", ""},
		{"one", "AGENT_INDEX=1\n"}, {"large", "AGENT_INDEX=20\n"},
		{"back4", "AGENT_INDEX=oops\n"},
	} {
		child := filepath.Join(t.TempDir(), tc.name)
		git("worktree", "add", "--detach", child)
		child, _ = NormalizePath(child)
		children[tc.name] = child
		if tc.env != "" {
			if err := os.WriteFile(filepath.Join(child, ".env.local"), []byte(tc.env), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	// A removed worktree's registration must not become a usable folder.
	gone := filepath.Join(t.TempDir(), "gone")
	git("worktree", "add", "--detach", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if _, err := s.SetVisible(repo, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetShortcut(repo, "A"); err != nil {
		t.Fatal(err)
	}
	// An existing user's choice must survive even if it differs from the index.
	if _, err := s.SetVisible(children["explicit"], true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetShortcut(children["explicit"], "z"); err != nil {
		t.Fatal(err)
	}
	d, err := s.Discover(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Folders) != len(children)+1 {
		t.Fatalf("folders: %+v", d.Folders)
	}
	byPath := map[string]Folder{}
	for _, f := range d.Folders {
		byPath[f.Path] = f
	}
	for name, want := range map[string]string{"explicit": "z", "back2": "11", "back19": "19", "implicit": "", "one": "1", "large": "", "back4": ""} {
		f := byPath[children[name]]
		if f.MenubarItemShortcut != want {
			t.Errorf("%s shortcut %q want %q", name, f.MenubarItemShortcut, want)
		}
		if name != "explicit" && (!f.Visible || !f.Discovered || f.WorktreeOf != repo) {
			t.Errorf("did not inherit monitoring: %+v", f)
		}
	}
	if _, err := s.SetVisible(children["back19"], false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetShortcut(children["back19"], ""); err != nil {
		t.Fatal(err)
	}
	d, err = s.Discover([]string{children["back19"], children["one"]})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range d.Folders {
		if f.Path == children["back19"] && (f.MenubarItemShortcut != "" || f.Visible) {
			t.Fatal("rediscovery overwrote explicit Hide")
		}
	}
	// Main checkout without a shortcut must not opt children into the menu bar.
	s = &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if _, err := s.SetVisible(repo, true); err != nil {
		t.Fatal(err)
	}
	d, err = s.Discover([]string{children["one"]})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range d.Folders {
		if f.MenubarItemShortcut != "" {
			t.Fatal("inherited without parent shortcut")
		}
	}
	s = &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if _, err := s.SetVisible(repo, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetVisible(repo, false); err != nil {
		t.Fatal(err)
	}
	d, err = s.Discover(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range d.Folders {
		if f.Visible {
			t.Fatal("enabled child of unmonitored main checkout")
		}
	}
}

func TestDiscoverRemovesOnlyMissingKnownWorktrees(t *testing.T) {
	repo := gitFolder(t)
	child := filepath.Join(t.TempDir(), "linked")
	for _, args := range [][]string{
		{"-C", repo, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-qm", "Initial"},
		{"-C", repo, "worktree", "add", "--detach", child},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	child, _ = NormalizePath(child)
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	// Simulate a pre-upgrade entry with no worktree provenance.
	_, err := s.update(func(d *Data) error {
		d.Folders = []Folder{{Path: repo, Visible: true}, {Path: child, Visible: true}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.Discover(nil)
	if err != nil || len(d.Folders) != 2 || d.Folders[1].WorktreeOf != repo {
		t.Fatalf("provenance: %+v %v", d, err)
	}
	if err := os.RemoveAll(child); err != nil {
		t.Fatal(err)
	}
	d, err = s.Discover(nil)
	if err != nil || len(d.Folders) != 1 || d.Folders[0].Path != repo {
		t.Fatalf("cleanup: %+v %v", d, err)
	}
	if err := os.RemoveAll(repo); err != nil {
		t.Fatal(err)
	}
	d, err = s.Discover(nil)
	if err != nil || len(d.Folders) != 1 {
		t.Fatalf("removed ordinary checkout: %+v %v", d, err)
	}
}
