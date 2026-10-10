package settings

import (
	"path/filepath"
	"testing"
)

func TestMarkdownAppearance(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if a := (MarkdownAppearance{}).Defaults(); a.Font != "github" || a.Size != 16 {
		t.Fatal(a)
	}
	if _, e := s.SetMarkdownAppearance(MarkdownAppearance{"iA Writer Duo S", 18}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SetMarkdownAppearance(MarkdownAppearance{"github", 50}); e == nil {
		t.Fatal("accepted invalid size")
	}
	d, e := s.Load()
	if e != nil || d.MarkdownAppearance.Font != "iA Writer Duo S" || d.MarkdownAppearance.Size != 18 {
		t.Fatal(d, e)
	}
}
