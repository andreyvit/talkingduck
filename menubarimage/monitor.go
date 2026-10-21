package menubarimage

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
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
	Mode       Mode
	Bars       []LimitBar
}

var monitorFontOnce sync.Once
var monitorFont *opentype.Font

func monitorFace(size float64) font.Face {
	monitorFontOnce.Do(func() { monitorFont, _ = opentype.Parse(gomono.TTF) })
	face, _ := opentype.NewFace(monitorFont, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	return face
}

// RenderMonitor matches the outer edges of the project grid at four pixels/point.
func RenderMonitor(m Monitor) *image.NRGBA {
	const scale = 4
	const barWidth = 26
	height := 23 * scale
	if m.Mode == Compact {
		height = 21 * scale
	}
	width := 0
	if m.Ping != "" {
		width = 8
	}
	if len(m.Bars) > 0 {
		if width > 0 {
			width += 3
		}
		width += barWidth
	}
	if width == 0 {
		return image.NewNRGBA(image.Rectangle{})
	}
	out := image.NewNRGBA(image.Rect(0, 0, width*scale, height))
	dark := color.NRGBA{24, 30, 38, 255}
	track := color.NRGBA{69, 77, 89, 255}
	white := color.NRGBA{255, 255, 255, 255}
	face := monitorFace(6 * scale)
	defer face.Close()
	if m.Ping != "" {
		pingFace := monitorFace(6.5 * scale)
		defer pingFace.Close()
		d := font.Drawer{Face: pingFace}
		w := d.MeasureString(m.Ping).Ceil()
		label := image.NewNRGBA(image.Rect(0, 0, w, 8*scale))
		d.Dst = label
		ink := white
		if m.PingFailed || m.PingMS >= 150 {
			ink = color.NRGBA{255, 174, 174, 255}
		} else if m.PingMS >= 50 {
			ink = color.NRGBA{255, 244, 174, 255}
		}
		d.Src = image.NewUniform(ink)
		d.Dot = fixed.P(0, 7*scale)
		d.DrawString(m.Ping)
		// Fix the unit's last ink to the top; only the number's bottom edge moves.
		last := 0
		for y := 0; y < label.Bounds().Dy(); y++ {
			for x := 0; x < w; x++ {
				if label.NRGBAAt(x, y).A > 0 {
					last = max(last, x)
				}
			}
		}
		span := min(last+1, height)
		for y := 0; y < label.Bounds().Dy(); y++ {
			for destY := 0; destY < span; destY++ {
				srcX := last - destY
				if last+1 > height {
					srcX = last - destY*(last+1)/height
				}
				c := label.NRGBAAt(srcX, y)
				if c.A > 0 {
					out.SetNRGBA(y, destY, c)
				}
			}
		}
	}
	n := min(3, len(m.Bars))
	x := 0
	if m.Ping != "" {
		x = 11 * scale
	}
	y := 0
	trackHeight := (height - (n-1)*scale) / max(1, n)
	if n == 1 {
		trackHeight = 7 * scale
		y = (height - trackHeight) / 2
	}
	for i, b := range m.Bars[:n] {
		bottom := y + trackHeight
		if n > 1 && i == n-1 {
			bottom = height
		}
		rect := image.Rect(x, y, x+barWidth*scale, bottom)
		draw.Draw(out, rect, image.NewUniform(track), image.Point{}, draw.Src)
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
		left, right := x, x+barWidth*scale
		ink := white
		if b.Remaining != nil && !math.IsNaN(*b.Remaining) && !math.IsInf(*b.Remaining, 0) {
			value := math.Max(0, math.Min(100, *b.Remaining))
			split := x + int(math.Round(barWidth*scale*value/100))
			draw.Draw(out, image.Rect(x, y, split, bottom), image.NewUniform(tint), image.Point{}, draw.Src)
			label = fmt.Sprintf("%.0f%%", value)
			if value > 50 {
				right = split
				ink = dark
			} else {
				left = split
			}
		}
		if b.Stale {
			label = "~" + label
		}
		d := font.Drawer{Dst: out, Src: image.NewUniform(ink), Face: face}
		tw := d.MeasureString(label).Ceil()
		if tw > right-left-2 {
			small := monitorFace(6 * scale * float64(right-left-2) / float64(tw))
			defer small.Close()
			d.Face = small
			tw = d.MeasureString(label).Ceil()
		}
		metrics := d.Face.Metrics()
		baseline := (y+bottom-metrics.Ascent.Ceil()-metrics.Descent.Ceil())/2 + metrics.Ascent.Ceil()
		d.Dot = fixed.P(left+(right-left-tw)/2, baseline)
		d.DrawString(label)
		// One-point corner cuts soften the track without extra padding.
		for dy := 0; dy < scale; dy++ {
			for dx := 0; dx < scale; dx++ {
				if dx+dy < scale-1 {
					out.SetNRGBA(x+dx, y+dy, color.NRGBA{})
					out.SetNRGBA(x+barWidth*scale-1-dx, y+dy, color.NRGBA{})
					out.SetNRGBA(x+dx, bottom-1-dy, color.NRGBA{})
					out.SetNRGBA(x+barWidth*scale-1-dx, bottom-1-dy, color.NRGBA{})
				}
			}
		}
		y = bottom + scale
	}
	return out
}

// JoinMonitor places the independent monitor immediately before project status.
func JoinMonitor(prefix, projects *image.NRGBA) *image.NRGBA {
	if prefix.Bounds().Empty() {
		return projects
	}
	if projects == nil || projects.Bounds().Empty() {
		return prefix
	}
	gap := 4 * 4
	h := max(prefix.Bounds().Dy(), projects.Bounds().Dy())
	out := image.NewNRGBA(image.Rect(0, 0, prefix.Bounds().Dx()+gap+projects.Bounds().Dx(), h))
	draw.Draw(out, prefix.Bounds().Add(image.Pt(0, (h-prefix.Bounds().Dy())/2)), prefix, image.Point{}, draw.Src)
	draw.Draw(out, projects.Bounds().Add(image.Pt(prefix.Bounds().Dx()+gap, (h-projects.Bounds().Dy())/2)), projects, image.Point{}, draw.Src)
	return out
}
