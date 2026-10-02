package cli

import (
	"io"
	"strings"
	"testing"
)

func TestCounterColors(t *testing.T) {
	for _, tc := range []struct {
		column        int
		zero, nonzero string
	}{
		{3, " 0", " 2"}, {9, "± 0", "±229"}, {10, "? 0", "? 5"},
		{11, "ℹ︎ 0", "ℹ︎ 4"}, {12, "⇣ 0", "⇣27"}, {13, "⇡ 0", "⇡ 2"},
	} {
		if !strings.Contains(colorCell(tc.zero, tc.column), "[38;5;242m") {
			t.Errorf("zero not gray: %q", tc.zero)
		}
		if strings.Contains(colorCell(tc.nonzero, tc.column), "[38;5;242m") {
			t.Errorf("nonzero gray: %q", tc.nonzero)
		}
	}
	pair := colorCell(" 0/24", 5)
	if !strings.Contains(pair, "\x1b[38;5;242m 0") || !strings.Contains(pair, "\x1b[38;5;75m24") {
		t.Fatal(pair)
	}
	if got := colorCell("title", 14); got != "\x1b[38;5;15mtitle\x1b[0m" {
		t.Fatal(got)
	}
	conflict := colorCell("Grok! ", 1)
	if !strings.Contains(conflict, "\x1b[38;5;203m!\x1b[0m") {
		t.Fatal(conflict)
	}
}

func TestColorOverrides(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if useColor("auto", io.Discard) || useColor("never", io.Discard) || !useColor("always", io.Discard) {
		t.Fatal("NO_COLOR or override ignored")
	}
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	if useColor("auto", io.Discard) {
		t.Fatal("dumb terminal colored")
	}
}
