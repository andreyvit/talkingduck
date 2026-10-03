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
