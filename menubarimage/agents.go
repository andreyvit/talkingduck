package menubarimage

import (
	"image"
	"image/color"
	"strconv"
)

// RenderAgents draws cropped agent motifs at four pixels per logical point.
// Motifs are simplified silhouettes, not official logo artwork.
func RenderAgents(cells []Cell, fg color.NRGBA, mode Mode) *image.NRGBA {
	if len(cells) == 0 {
		return image.NewNRGBA(image.Rectangle{})
	}
	base := Render(cells, fg, mode)

	cw, ch, gap := 23, 11, 2
	if mode == Compact {
		cw, ch, gap = 19, 10, 1
	}
	cols := (len(cells) + 1) / 2
	oldW := cw
	advance, glyphH := 6, 7
	if mode == Compact {
		advance, glyphH = 5, 6
	}
	widths, positions := make([]int, len(cells)), make([]int, len(cells))
	total, rowWidth := 0, 0
	labelWidth := agentLabelWidth(10, advance)
	for _, c := range cells {
		labelWidth = max(labelWidth, shortcutLabelWidth(c.shortcut(), advance))
	}
	for i := range cells {
		if i == cols {
			total = rowWidth - gap
			rowWidth = 0
		}
		widths[i] = labelWidth + 3 + 2 + 5
		positions[i] = rowWidth
		rowWidth += widths[i] + gap
	}
	total = max(total, rowWidth-gap)
	expanded := image.NewNRGBA(image.Rect(0, 0, total, ch*2+1))
	for i, c := range cells {
		ox, oy := (i%cols)*(oldW+gap), (i/cols)*(ch+1)
		nx := positions[i]
		cw := widths[i]
		for y := 0; y < ch; y++ {
			for x := 0; x < cw; x++ {
				var ink color.NRGBA
				if c.State == "attention" {
					ink = fg
				} else if y == 0 || y == ch-1 {
					ink = base.NRGBAAt(ox+x%oldW, oy+y)
				} else if x == 0 || x == cw-1 {
					sx := 0
					if x > 0 {
						sx = oldW - 1
					}
					ink = base.NRGBAAt(ox+sx, oy+y)
				}
				expanded.SetNRGBA(nx+x, oy+y, ink)
			}
		}
		width := (len(c.shortcut())+1)*advance - 1
		tx := (oldW - width) / 2
		number := c.shortcut()
		label := number + c.Stage
		dest := nx + 3
		for j, r := range label {
			narrow := len(number) > 1 && j < len(number) && r == '1'
			glyphWidth := advance - 1
			if narrow {
				glyphWidth = 1
			}
			for y := 0; y < glyphH; y++ {
				for x := 0; x < glyphWidth; x++ {
					ink := base.NRGBAAt(ox+tx+j*advance+x, oy+2+y)
					if narrow {
						ink = fg
						if c.State == "attention" {
							ink = color.NRGBA{}
						} else if c.State == "idle" {
							ink.A = 115
						}
					}
					expanded.SetNRGBA(dest+x, oy+2+y, ink)
				}
			}
			dest += glyphWidth + 1
		}
	}
	out := Scale(expanded, 4)

	for i, c := range cells {
		cw := widths[i]
		ox, oy := positions[i], (i/cols)*(ch+1)
		for py := 4; py < (ch-1)*4; py++ {
			for px := (cw - 5) * 4; px < (cw-1)*4; px++ {
				// Draw on the logical pixel grid, then scale exactly: no hairlines
				// or overlapping antialiased curves that collapse into solid blobs.
				motif := agentPixels[c.Agent]
				mx, my := px/4-(cw-5), py/4-1
				if mode == Compact && my >= 4 {
					my++ // Omit the center row to preserve vertical symmetry at 8 pixels.
				}
				hit := my >= 0 && my < len(motif) && mx >= 0 && mx < 5 && motif[my][mx] == '1'

				if len(motif) == 0 {
					continue
				}
				// Keep the outer border intact; crop the motif inside it.
				ink := color.NRGBA{}
				if hit != (c.State == "attention") {
					ink = fg
					ink.A = 255
				}
				out.SetNRGBA(ox*4+px, oy*4+py, ink)
			}
		}
	}
	return out
}

// Only digit 1 inside a multi-digit number gets the clock-style narrow glyph.
func agentLabelWidth(number, advance int) int {
	return shortcutLabelWidth(strconv.Itoa(number), advance)
}
func shortcutLabelWidth(digits string, advance int) int {
	width := (len(digits)+1)*advance - 1
	if len(digits) > 1 {
		for _, r := range digits {
			if r == '1' {
				width -= advance - 2
			}
		}
	}
	return width
}

// Cropped, deliberately simplified agent silhouettes. Negative space is part of
// the design; each stroke is a full logical pixel at both display densities.
var agentPixels = map[string][]string{
	"claude": {
		"01001",
		"00101",
		"10010",
		"01111",
		"00011",
		"01111",
		"10010",
		"00101",
		"01001",
	},
	"codex": {
		"00111",
		"01100",
		"11001",
		"10001",
		"01111",
		"10001",
		"11001",
		"01100",
		"00111",
	},
	"grok": {
		"00111",
		"01000",
		"10001",
		"10011",
		"10110",
		"01100",
		"11000",
		"11000",
		"10111",
	},
}
