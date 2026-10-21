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
	Bars       []LimitBar
}

var monitorFontOnce sync.Once
var monitorFont *opentype.Font

func monitorFace(size float64) font.Face {
	monitorFontOnce.Do(func() { monitorFont, _ = opentype.Parse(gomono.TTF) })
	face, _ := opentype.NewFace(monitorFont, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	return face
}

// RenderMonitor uses four backing pixels per point, matching project status art.
// Three 6-point tracks + two 1-point gaps fit a 22-point menu bar.
func RenderMonitor(m Monitor) *image.NRGBA {
	const scale = 4
	width := 0
	if m.Ping != "" {
		width = 8
	}
	if len(m.Bars) > 0 {
		if width > 0 {
			width += 3
		}
		width += 42
	}
	if width == 0 {
		return image.NewNRGBA(image.Rectangle{})
	}
	out := image.NewNRGBA(image.Rect(0, 0, width*scale, 22*scale))
	dark := color.NRGBA{24, 30, 38, 255}
	track := color.NRGBA{69, 77, 89, 255}
	white := color.NRGBA{245, 247, 250, 255}
	face := monitorFace(6 * scale)
	defer face.Close()
	if m.Ping != "" {
		pingFace := monitorFace(5.5 * scale)
		defer pingFace.Close()
		d := font.Drawer{Face: pingFace}
		w := d.MeasureString(m.Ping).Ceil()
		label := image.NewNRGBA(image.Rect(0, 0, w, 7*scale))
		d.Dst = label
		ink := white
		if m.PingFailed {
			ink = color.NRGBA{255, 199, 97, 255}
		}
		d.Src = image.NewUniform(ink)
		d.Dot = fixed.P(0, 6*scale)
		d.DrawString(m.Ping)
		// Rotated bottom-to-top; native tray adapts normal text to menu-bar appearance.
		for y := 0; y < label.Bounds().Dy(); y++ {
			for x := 0; x < label.Bounds().Dx(); x++ {
				c := label.NRGBAAt(x, y)
				if c.A > 0 {
					out.SetNRGBA(y, (22*scale+w)/2-x-1, c)
				}
			}
		}
	}
	n := min(3, len(m.Bars))
	x := 0
	if m.Ping != "" {
		x = 11 * scale
	}
	y := (22*scale - (n*6+(n-1))*scale) / 2
	for _, b := range m.Bars[:n] {
		rect := image.Rect(x, y, x+42*scale, y+6*scale)
		draw.Draw(out, rect, image.NewUniform(track), image.Point{}, draw.Src)
		tint := white
		if n > 1 {
			switch b.ID {
			case "codex":
				tint = color.NRGBA{201, 241, 230, 255}
			case "claude":
				tint = color.NRGBA{255, 219, 181, 255}
			case "grok":
				tint = color.NRGBA{216, 207, 255, 255}
			}
		}
		label := "—"
		left, right := x, x+42*scale
		ink := white
		if b.Remaining != nil && !math.IsNaN(*b.Remaining) && !math.IsInf(*b.Remaining, 0) {
			value := math.Max(0, math.Min(100, *b.Remaining))
			split := x + int(math.Round(42*scale*value/100))
			draw.Draw(out, image.Rect(x, y, split, y+6*scale), image.NewUniform(tint), image.Point{}, draw.Src)
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
		d.Dot = fixed.P(left+(right-left-tw)/2, y+5*scale)
		d.DrawString(label)
		y += 7 * scale
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
