package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGroups(t *testing.T) {
	for _, tc := range []struct {
		path         string
		product, doc bool
	}{
		{"main.go", true, false}, {"App/State.swift", true, false},
		{"main_test.go", true, false}, {"scripts/scout/scout.go", false, false},
		{"scripts/scout/scout_test.go", false, false}, {"docs/product.md", false, true},
		{"README.md", false, true}, {"AGENTS.md", false, true}, {".zoo/review.md", false, true},
		{".spec/task.md", false, false}, {"images/duck.png", false, false},
		{"mydocs/test.md", false, false}, {"README.md.old", false, false},
	} {
		if productFile(tc.path) != tc.product || docFile(tc.path) != tc.doc {
			t.Errorf("groups for %s", tc.path)
		}
	}
}

func TestModeGates(t *testing.T) {
	changes := []change{{path: "main.go"}, {path: "docs/product.md"}, {path: "scripts/scout/scout.go"}}
	for _, opts := range []options{{highlevel: true}, {plan: true}, {plan: true, uber: true}, {highlevel: true, uber: true}} {
		items, err := questions(changes, opts)
		if err != nil || strings.Contains(strings.Join(items, "\n"), "[folder-context]") {
			t.Fatalf("planning gated on implementation: %v, %v", items, err)
		}
		wantSplit := opts.plan
		if strings.Contains(strings.Join(items, "\n"), "subtask split") != wantSplit {
			t.Fatal("low-level question in wrong mode")
		}
		if strings.Contains(strings.Join(items, "\n"), "integration and rollout risks") != opts.uber {
			t.Fatal("planning uber coverage in wrong mode")
		}
	}
	items, err := questions(changes, options{selected: []string{"folder-context"}})
	if err != nil || len(items) != 1 || !strings.HasPrefix(items[0], "main.go - ") {
		t.Fatalf("wrong file gate: %v, %v", items, err)
	}
	items, err = questions([]change{{path: "renamed.txt", oldPath: "old.swift"}}, options{selected: []string{"state-evidence"}})
	if err != nil || len(items) != 1 {
		t.Fatal("rename lost original file gate")
	}
	items, err = questions(nil, options{selected: []string{"folder-context"}})
	if err != nil || len(items) != 0 {
		t.Fatal("unmatched group produced questions")
	}
	if _, err := questions(nil, options{selected: []string{"bogus"}}); err == nil {
		t.Fatal("unknown check accepted")
	}
	items, err = questions(nil, options{uber: true})
	if err != nil || len(items) != len(codeQuestions)+1 {
		t.Fatal("missing whole-change question")
	}
	items, err = questions(changes, options{uber: true})
	if err != nil || !strings.HasPrefix(items[len(codeQuestions)], "Does the entire change") || !strings.Contains(items[len(codeQuestions)+1], "[folder-context]") {
		t.Fatal("plain uber question must precede gated items")
	}
}

func TestNameStatus(t *testing.T) {
	got, err := parseChanges([]byte("M\x00space name.go\x00R100\x00old.swift\x00new.swift\x00D\x00gone.go\x00"))
	want := []change{{path: "space name.go"}, {path: "new.swift", oldPath: "old.swift"}, {path: "gone.go"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("%v, %v", got, err)
	}
	for _, malformed := range []string{"M\x00", "R100\x00old.go\x00"} {
		if _, err := parseChanges([]byte(malformed)); err == nil {
			t.Fatalf("accepted malformed record %q", malformed)
		}
	}
}

func repo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := t.TempDir()
	gitOK(t, root, "init", "-b", "main")
	gitOK(t, root, "config", "user.name", "Scout Test")
	gitOK(t, root, "config", "user.email", "scout@example.invalid")
	write(t, root, ".gitignore", ".tasks/\n")
	write(t, root, "main.go", "package main\n")
	write(t, root, "old.swift", "// original\n")
	gitOK(t, root, "add", ".")
	gitOK(t, root, "commit", "-m", "baseline")
	return root
}

func gitOK(t *testing.T, root string, args ...string) string {
	t.Helper()
	out, err := git(root, args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, root, path, body string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGitScopes(t *testing.T) {
	root := repo(t)
	base := gitOK(t, root, "rev-parse", "HEAD")
	gitOK(t, root, "mv", "old.swift", "new.swift")
	gitOK(t, root, "commit", "-m", "rename")
	write(t, root, "main.go", "package main\n// edit\n")
	write(t, root, "folder/space name.swift", "// new\n")
	write(t, root, ".tasks/private.md", "ignored\n")
	got, err := loadChanges(root, "")
	want := []change{{path: "folder/space name.swift"}, {path: "main.go"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("default scope: %v, %v", got, err)
	}
	got, err = loadChanges(root, base)
	want = append(want, change{path: "new.swift", oldPath: "old.swift"})
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("since scope: %v, %v", got, err)
	}
	gitOK(t, root, "update-ref", "refs/remotes/origin/main", base)
	gitOK(t, root, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	if gotBase, err := baseRef(root, "upstream"); err != nil || gotBase != base {
		t.Fatalf("fallback base: %s, %v", gotBase, err)
	}
	gitOK(t, root, "config", "branch.main.remote", "origin")
	gitOK(t, root, "config", "branch.main.merge", "refs/heads/main")
	if gotBase, err := baseRef(root, "upstream"); err != nil || gotBase != base {
		t.Fatalf("tracking base: %s, %v", gotBase, err)
	}
	if _, err := loadChanges(root, "missing"); err == nil {
		t.Fatal("missing ref accepted")
	}
	if err := os.Remove(filepath.Join(root, "main.go")); err != nil {
		t.Fatal(err)
	}
	got, err = loadChanges(root, "")
	if err != nil || !reflect.DeepEqual(got, []change{{path: "folder/space name.swift"}, {path: "main.go"}}) {
		t.Fatal("deleted source lost from scope")
	}
}

func TestCLIErrorsAndOutput(t *testing.T) {
	root := repo(t)
	t.Chdir(root)
	for _, args := range [][]string{{"-plan", "-highlevel-plan"}, {"-since", "missing"}, {"unknown"}} {
		var stdout, stderr bytes.Buffer
		if err := run(args, &stdout, &stderr); err == nil || stdout.Len() != 0 {
			t.Fatalf("partial success for %v: %s", args, &stdout)
		}
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"folder-context"}, &stdout, &stderr); err != nil || stdout.String() != "NO QUESTIONS\n" {
		t.Fatalf("empty output: %s, %v", &stdout, err)
	}
	write(t, root, "new.go", "package main\n")
	stdout.Reset()
	if err := run([]string{"-v", "folder-context"}, &stdout, &stderr); err != nil || !strings.Contains(stdout.String(), "1. new.go - ") || !strings.Contains(stderr.String(), "folder-context: 1 spots") {
		t.Fatalf("gated output: %s, %s, %v", &stdout, &stderr, err)
	}
	stdout.Reset()
	t.Chdir(t.TempDir())
	if err := run(nil, &stdout, &stderr); err == nil || stdout.Len() != 0 {
		t.Fatal("non-repo accepted")
	}
}
