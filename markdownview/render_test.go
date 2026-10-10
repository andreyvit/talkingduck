package markdownview

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderGFMAndZoo(t *testing.T) {
	source := []byte("---\nstatus: executing\nnested:\n  answer: 42\nunsafe: '<script>alert(1)</script>'\n---\n# Hello *world*\n\n# Hello world\n\n# Hello world-1\n\n| A | B |\n|---|---|\n| one | two |\n\n~~gone~~ https://example.com\n\n1. [ ] Pending\n2. [>] Working\n3. [x] Done\n\n### [>] Heading task\n\n```md\n- [>] literal\n```\n")
	d, e := Render(source, "/work/spec.md")
	if e != nil {
		t.Fatal(e)
	}
	for _, want := range []string{`class="frontmatter"`, `<th>status</th>`, `answer: 42`, `&lt;script&gt;`, `<table>`, `<del>gone</del>`, `href="https://example.com"`, `type="checkbox"`, `checked=""`, `task-status running`, `- [&gt;] literal`} {
		if !strings.Contains(d.HTML, want) {
			t.Errorf("missing %q in %s", want, d.HTML)
		}
	}
	if len(d.Headings) != 4 || d.Headings[0].Title != "Hello world" || d.Headings[1].ID != "hello-world-1" || d.Headings[2].ID != "hello-world-1-1" {
		t.Fatal(d.Headings)
	}
	if strings.Count(d.HTML, `task-status running`) != 2 {
		t.Fatal(d.HTML)
	}
	if len(d.Warnings) != 0 {
		t.Fatal(d.Warnings)
	}
}
func TestImagesAndUntrustedHTML(t *testing.T) {
	dir := t.TempDir()
	image := filepath.Join(dir, "picture space.png")
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if e := os.WriteFile(image, png, 0600); e != nil {
		t.Fatal(e)
	}
	source := []byte("![relative](picture%20space.png)\n![absolute](<" + image + ">)\n<img src=\"picture%20space.png\" onerror=\"alert(1)\">\n\n<script>alert(2)</script><iframe src=\"https://evil.test\"></iframe><a href=\"javascript:alert(3)\">bad</a>\n")
	d, e := Render(source, filepath.Join(dir, "spec.md"))
	if e != nil {
		t.Fatal(e)
	}
	if strings.Count(d.HTML, "data:image/png;base64,") != 3 {
		t.Fatal(d.HTML, d.Warnings)
	}
	for _, bad := range []string{"onerror", "<script", "<iframe", "javascript:", "alert("} {
		if strings.Contains(d.HTML, bad) {
			t.Fatalf("unsafe %s: %s", bad, d.HTML)
		}
	}
	d, e = Render([]byte("![missing](gone.png)"), filepath.Join(dir, "spec.md"))
	if e != nil || len(d.Warnings) != 1 {
		t.Fatal(d, e)
	}
}
func TestFrontmatterFailuresAndCode(t *testing.T) {
	for _, source := range []string{"---\nbad: [\n---\n# Body", "---\n- list\n---\n# Body", "---\nnot closed"} {
		d, e := Render([]byte(source), "/spec.md")
		if e != nil || len(d.Warnings) != 1 || d.HTML == "" {
			t.Fatal(d, e)
		}
	}
	d, e := Render([]byte("---\r\na: true\r\n---\r\n# Body\r\n\n`[>] literal`\n\n    - [>] code\n"), "/spec.md")
	if e != nil || len(d.Headings) != 1 || strings.Contains(d.HTML, "task-status") {
		t.Fatal(d, e)
	}
}
func TestCheckboxState(t *testing.T) {
	d, e := Render([]byte("- [ ] Pending\n- [x] Done\n"), "/spec.md")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Count(d.HTML, `checked=""`) != 1 || strings.Count(d.HTML, `disabled=""`) != 2 {
		t.Fatal(d.HTML)
	}
}
func TestReadFileLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.md")
	os.WriteFile(path, []byte("12345"), 0600)
	if _, e := ReadFile(path, 4); e == nil {
		t.Fatal("accepted oversized file")
	}
	if _, e := ReadFile(filepath.Dir(path), 4); e == nil {
		t.Fatal("accepted directory")
	}
}
