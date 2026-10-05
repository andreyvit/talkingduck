package menubarimage

import (
	"bytes"
	"image/color"
	"testing"
)

func TestClockStyleOnesOnlyInMultiDigitNumbers(t *testing.T) {
	for _, tc := range []struct{ number, advance, width int }{
		{1, 6, 11}, {10, 6, 13}, {11, 6, 9}, {21, 6, 13}, {22, 6, 17},
		{1, 5, 9}, {10, 5, 11}, {11, 5, 8},
	} {
		if got := agentLabelWidth(tc.number, tc.advance); got != tc.width {
			t.Errorf("number %d advance %d: got %d want %d", tc.number, tc.advance, got, tc.width)
		}
	}
}

func TestAgentMotifsAndEqualGrid(t *testing.T) {
	fg := color.NRGBA{A: 255}
	cells := []Cell{{Number: 1, Stage: "H", State: "running"}, {Number: 10, Stage: "H", State: "running"}}
	base := RenderAgents(cells, fg, Standard)
	if base.Bounds().Dx() != 23*4 || base.Bounds().Dy() != 23*4 {
		t.Fatal(base.Bounds())
	}
	previous := map[string][]byte{}
	for _, agent := range []string{"claude", "codex", "grok"} {
		cells[0].Agent = agent
		got := RenderAgents(cells, fg, Standard)
		if bytes.Equal(got.Pix, base.Pix) {
			t.Fatal("missing motif", agent)
		}
		for other, pixels := range previous {
			if bytes.Equal(got.Pix, pixels) {
				t.Fatal("identical motifs", agent, other)
			}
		}
		previous[agent] = got.Pix
	}
	if !RenderAgents(nil, fg, Standard).Bounds().Empty() {
		t.Fatal("nonempty")
	}
}

func TestAgentStrokesStayOnLogicalPixels(t *testing.T) {
	for _, mode := range []Mode{Standard, Compact} {
		for agent := range agentPixels {
			im := RenderAgents([]Cell{{Number: 10, Stage: "H", State: "attention", Agent: agent}}, color.NRGBA{A: 255}, mode)
			for y := 0; y < im.Bounds().Dy(); y++ {
				for x := 0; x < im.Bounds().Dx(); x++ {
					a := im.NRGBAAt(x, y).A
					if a != 0 && a != 255 {
						t.Fatal("partial coverage", mode, agent, x, y, a)
					}
					if im.NRGBAAt(x, y) != im.NRGBAAt(x/4*4, y/4*4) {
						t.Fatal("subpixel stroke", mode, agent, x, y)
					}
				}
			}
		}
	}
}

// Outlined and filled cells must show the same motif with opposite polarity.
func TestAgentMotifPolarity(t *testing.T) {
	for _, mode := range []Mode{Standard, Compact} {
		for agent := range agentPixels {
			cell := Cell{Number: 10, Stage: "H", Agent: agent, State: "running"}
			normal := RenderAgents([]Cell{cell}, color.NRGBA{A: 255}, mode)
			cell.State = "attention"
			filled := RenderAgents([]Cell{cell}, color.NRGBA{A: 255}, mode)
			height := 11
			if mode == Compact {
				height = 10
			}
			for y := 4; y < (height-1)*4; y++ {
				for x := normal.Bounds().Dx() - 20; x < normal.Bounds().Dx()-4; x++ {
					if int(normal.NRGBAAt(x, y).A)+int(filled.NRGBAAt(x, y).A) != 255 {
						t.Fatalf("motif changed with state: %v %s at %d,%d", mode, agent, x, y)
					}
				}
			}
		}
	}
}

// Agent artwork must not erase or brighten the cell's status border.
func TestAgentPreservesRightBorder(t *testing.T) {
	for _, mode := range []Mode{Standard, Compact} {
		for _, state := range []string{"running", "idle", "attention", "unknown"} {
			cell := Cell{Number: 10, Stage: "H", State: state}
			base := RenderAgents([]Cell{cell}, color.NRGBA{A: 255}, mode)
			for agent := range agentPixels {
				cell.Agent = agent
				got := RenderAgents([]Cell{cell}, color.NRGBA{A: 255}, mode)
				for y := 0; y < got.Bounds().Dy(); y++ {
					for x := got.Bounds().Dx() - 4; x < got.Bounds().Dx(); x++ {
						if got.NRGBAAt(x, y) != base.NRGBAAt(x, y) {
							t.Fatalf("right border changed: %v %s %s at %d,%d", mode, state, agent, x, y)
						}
					}
				}
			}
		}
	}
}

func TestShortcutGlyphCoverage(t *testing.T) {
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789" {
		if len(standardGlyphs[r]) != 7 || len(compactGlyphs[r]) != 6 {
			t.Fatalf("missing glyph %c", r)
		}
	}
	for _, label := range []string{"A", "a", "0", "10", "11", "19"} {
		for _, mode := range []Mode{Standard, Compact} {
			im := RenderAgents([]Cell{{Shortcut: label, Stage: "H", State: "running"}}, color.NRGBA{A: 255}, mode)
			if im.Bounds().Dx() == 0 {
				t.Fatal(label, mode)
			}
		}
	}
}
