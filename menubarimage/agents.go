package menubarimage

import (
	"image"
	"image/color"
	"math"
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
		labelWidth = max(labelWidth, agentLabelWidth(c.Number, advance))
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
		width := (len(strconv.Itoa(c.Number))+1)*advance - 1
		tx := (oldW - width) / 2
		number := strconv.Itoa(c.Number)
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
		for py := 1; py < (ch-1)*4; py++ {
			for px := (cw - 5) * 4; px < cw*4; px++ {
				x, y := float64(px)/4-float64(cw)+.4, float64(py)/4-float64(ch)/2
				hit := false
				switch c.Agent {
				case "claude": // Claude: radial, slightly uneven sunburst.
					r, a := math.Hypot(x, y), math.Atan2(y, x)
					hit = r < 4.7 && r > 1 && math.Abs(math.Sin(a*6)) < .23
				case "codex": // Codex/OpenAI-inspired interlocking loops.
					for n := 0; n < 6; n++ {
						a := float64(n) * math.Pi / 3
						r := math.Hypot(x-2.15*math.Cos(a), y-2.15*math.Sin(a))
						hit = hit || math.Abs(r-2.4) < .33
					}
				case "grok": // Grok: ring with a diagonal slash.
					hit = math.Abs(math.Hypot(x, y)-3.8) < .4 || (math.Abs(x+y) < .48 && math.Abs(y) < 4.7)
				}
				if !hit {
					continue
				}
				xx, yy := ox*4+px, oy*4+py
				old := out.NRGBAAt(xx, yy)
				if c.State == "attention" {
					if old.A == 255 {
						old.A = 185
						out.SetNRGBA(xx, yy, old)
					}
				} else if old.A == 0 {
					ink := fg
					ink.A = 85
					out.SetNRGBA(xx, yy, ink)
				}
			}
		}
	}
	return out
}

// Only digit 1 inside a multi-digit number gets the clock-style narrow glyph.
func agentLabelWidth(number, advance int) int {
	digits := strconv.Itoa(number)
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
