package main

import (
	"image"
	"image/color"
)

// Average premultiplied backing pixels at the target display resolution first.
// Preview enlargement then uses nearest-neighbor, exposing those actual pixels.
func downsample(src image.Image, factor int) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()/factor, b.Dy()/factor))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			var r, g, bl, a uint32
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					rr, gg, bb, aa := src.At(b.Min.X+x*factor+dx, b.Min.Y+y*factor+dy).RGBA()
					r += rr
					g += gg
					bl += bb
					a += aa
				}
			}
			n := uint32(factor * factor)
			out.Set(x, y, color.RGBA64{uint16(r / n), uint16(g / n), uint16(bl / n), uint16(a / n)})
		}
	}
	return out
}
