package main

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"talkingduck/menubarimage"
)

// Experimental cropped silhouettes, not official logo artwork. The agent order
// repeats Claude, Codex, Grok; the production tray renderer is unchanged.
func agentBackdrop(base *image.NRGBA, cells []menubarimage.Cell, mode menubarimage.Mode, fg color.NRGBA) *image.NRGBA {

	cw, ch, gap := 23, 11, 2
	if mode == menubarimage.Compact {
		cw, ch, gap = 19, 10, 1
	}
	cols := (len(cells) + 1) / 2
	oldW := cw
	advance, glyphH := 6, 7
	if mode == menubarimage.Compact {
		advance, glyphH = 5, 6
	}
	widths, positions := make([]int, len(cells)), make([]int, len(cells))
	total, rowWidth := 0, 0
	for i, c := range cells {
		if i == cols {
			total = rowWidth - gap
			rowWidth = 0
		}
		widths[i] = (len(strconv.Itoa(c.Number))+1)*advance - 1 + 1 + 1 + 5
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
		for y := 0; y < glyphH; y++ {
			for x := 0; x < width; x++ {
				expanded.SetNRGBA(nx+1+x, oy+2+y, base.NRGBAAt(ox+tx+x, oy+2+y))
			}
		}
	}
	out := menubarimage.Scale(expanded, 4)

	for i, c := range cells {
		cw := widths[i]
		ox, oy := positions[i], (i/cols)*(ch+1)
		for py := 1; py < (ch-1)*4; py++ {
			for px := (cw - 5) * 4; px < cw*4; px++ {
				x, y := float64(px)/4-float64(cw)+.4, float64(py)/4-float64(ch)/2
				hit := false
				switch i % 3 {
				case 0: // Claude: radial, slightly uneven sunburst.
					r, a := math.Hypot(x, y), math.Atan2(y, x)
					hit = r < 4.7 && r > 1 && math.Abs(math.Sin(a*6)) < .23
				case 1: // Codex/OpenAI-inspired interlocking loops.
					for n := 0; n < 6; n++ {
						a := float64(n) * math.Pi / 3
						r := math.Hypot(x-2.15*math.Cos(a), y-2.15*math.Sin(a))
						hit = hit || math.Abs(r-2.4) < .33
					}
				case 2: // Grok: ring with a diagonal slash.
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
