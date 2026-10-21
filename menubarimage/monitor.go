package menubarimage

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
)

type LimitBar struct {
	ID        string
	Remaining *float64
	Stale     bool
}
type Monitor struct {
	Ping       string
	PingFailed bool
	PingMS     float64
	PingDigits int
	Mode       Mode
	Bars       []LimitBar
}

// Tiny digits complement the shared 4x6 font on narrow quota tracks.
var tinyDigits = map[rune][]string{
	'0': {"111", "101", "101", "101", "111"}, '1': {"010", "110", "010", "010", "111"},
	'2': {"110", "001", "010", "100", "111"}, '3': {"110", "001", "010", "001", "110"},
	'4': {"101", "101", "111", "001", "001"}, '5': {"111", "100", "110", "001", "110"},
	'6': {"011", "100", "111", "101", "111"}, '7': {"111", "001", "010", "010", "010"},
	'8': {"111", "101", "111", "101", "111"}, '9': {"111", "101", "111", "001", "110"},
	'm': {"101", "111", "111", "101", "101"}, 's': {"011", "100", "010", "001", "110"},
	'—': {"000", "000", "111", "000", "000"}, '…': {"000", "000", "000", "000", "101"},
	'!': {"010", "010", "010", "000", "010"}, '~': {"000", "010", "101", "000", "000"},
	'+': {"000", "010", "111", "010", "000"},
}

// Text is built on a logical-point grid and enlarged only by an integer scale.
func monitorText(text string, compact bool) *image.NRGBA {
	glyphs := [][]string{}
	width := 0
	for _, r := range text {
		if r == ' ' {
			glyphs = append(glyphs, nil)
			width++
			continue
		}
		glyph := tinyDigits[r]
		if compact && r >= '0' && r <= '9' {
			glyph = compactGlyphs[r]
		}
		if len(glyph) == 0 {
			glyph = tinyDigits['—']
		}
		glyphs = append(glyphs, glyph)
		width += len(glyph[0]) + 1
	}
	width = max(0, width-1)
	height := 5
	if compact {
		height = 6
	}
	im := image.NewNRGBA(image.Rect(0, 0, width, height))
	x := 0
	for i, g := range glyphs {
		if g == nil {
			x++
			continue
		}
		for y, row := range g {
			for xx, v := range row {
				if v == '1' {
					im.SetNRGBA(x+xx, height-len(g)+y, color.NRGBA{255, 255, 255, 255})
				}
			}
		}
		x += len(g[0])
		if i+1 < len(glyphs) && glyphs[i+1] != nil {
			x++
		}
	}
	// Width above includes no extra kerning around the single-point word space.
	if len(glyphs) > 0 {
		width = min(width, x)
	}
	return im.SubImage(image.Rect(0, 0, width, height)).(*image.NRGBA)
}
func RenderMonitor(m Monitor) *image.NRGBA {
	const barWidth = 17
	height := 23
	if m.Mode == Compact {
		height = 21
	}
	width := 0
	if m.Ping != "" {
		width = 5
	}
	if len(m.Bars) > 0 {
		if width > 0 {
			width += 2
		}
		width += barWidth
	}
	if width == 0 {
		return image.NewNRGBA(image.Rectangle{})
	}
	out := image.NewNRGBA(image.Rect(0, 0, width, height))
	dark := color.NRGBA{24, 30, 38, 255}
	track := color.NRGBA{69, 77, 89, 255}
	white := color.NRGBA{255, 255, 255, 255}
	if m.Ping != "" {
		// Use the quota font, with the rolling digit field centered vertically.
		digits := max(1, min(4, m.PingDigits))
		reserved := monitorText(strings.Repeat("0", digits), false).Bounds().Dx()
		label := monitorText(m.Ping, false)
		if m.PingFailed {
			reserved = label.Bounds().Dx()
		}
		reserved = max(reserved, label.Bounds().Dx())
		ink := white
		if m.PingFailed || m.PingMS >= 150 {
			ink = color.NRGBA{255, 174, 174, 255}
		} else if m.PingMS >= 50 {
			ink = color.NRGBA{255, 244, 174, 255}
		}
		offset := max(0, (height-reserved)/2)
		for y := 0; y < label.Bounds().Dy(); y++ {
			for x := 0; x < label.Bounds().Dx(); x++ {
				if label.NRGBAAt(x, y).A > 0 {
					out.SetNRGBA(y, offset+label.Bounds().Dx()-1-x, ink)
				}
			}
		}
	}
	n := min(3, len(m.Bars))
	x := 0
	if m.Ping != "" {
		x = 7
	}
	y := 0
	trackHeight := (height - (n - 1)) / max(1, n)
	if n == 1 {
		trackHeight = 7
		y = (height - trackHeight) / 2
	}
	for i, b := range m.Bars[:n] {
		bottom := y + trackHeight
		if n > 1 && i == n-1 {
			bottom = height
		}
		draw.Draw(out, image.Rect(x, y, x+barWidth, bottom), image.NewUniform(track), image.Point{}, draw.Src)
		tint := white
		if n > 1 {
			switch b.ID {
			case "codex":
				tint = color.NRGBA{239, 255, 250, 255}
			case "claude":
				tint = color.NRGBA{255, 248, 236, 255}
			case "grok":
				tint = color.NRGBA{248, 245, 255, 255}
			}
		}
		label := "—"
		left, right, split := x, x+barWidth, x
		if b.Remaining != nil && !math.IsNaN(*b.Remaining) && !math.IsInf(*b.Remaining, 0) {
			value := math.Max(0, math.Min(100, *b.Remaining))
			split = x + int(math.Round(barWidth*value/100))
			draw.Draw(out, image.Rect(x, y, split, bottom), image.NewUniform(tint), image.Point{}, draw.Src)
			label = fmt.Sprintf("%.0f", value)
			if value > 50 {
				right = split
			} else {
				left = split
			}
		}
		if b.Stale {
			label = "~" + label
		}
		glyph := monitorText(label, false)
		tw := glyph.Bounds().Dx()
		if tw > right-left {
			left, right = x, x+barWidth
		}
		tx, ty := left+(right-left-tw)/2, y+(bottom-y-5)/2
		for yy := 0; yy < 5; yy++ {
			for xx := 0; xx < tw; xx++ {
				if glyph.NRGBAAt(xx, yy).A > 0 {
					ink := white
					if tx+xx < split {
						ink = dark
					}
					out.SetNRGBA(tx+xx, ty+yy, ink)
				}
			}
		}
		for _, pt := range []image.Point{{x, y}, {x + barWidth - 1, y}, {x, bottom - 1}, {x + barWidth - 1, bottom - 1}} {
			out.SetNRGBA(pt.X, pt.Y, color.NRGBA{})
		}
		y = bottom + 1
	}
	return Scale(out, 4)
}

// JoinMonitor places the independent monitor immediately before project status.
func JoinMonitor(prefix, projects *image.NRGBA) *image.NRGBA {
	if prefix.Bounds().Empty() {
		return projects
	}
	if projects == nil || projects.Bounds().Empty() {
		return prefix
	}
	gap := 2 * 4
	h := max(prefix.Bounds().Dy(), projects.Bounds().Dy())
	out := image.NewNRGBA(image.Rect(0, 0, prefix.Bounds().Dx()+gap+projects.Bounds().Dx(), h))
	draw.Draw(out, prefix.Bounds().Add(image.Pt(0, (h-prefix.Bounds().Dy())/2)), prefix, image.Point{}, draw.Src)
	draw.Draw(out, projects.Bounds().Add(image.Pt(prefix.Bounds().Dx()+gap, (h-projects.Bounds().Dy())/2)), projects, image.Point{}, draw.Src)
	return out
}
