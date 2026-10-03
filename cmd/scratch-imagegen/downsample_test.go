package main

import (
	"image"
	"image/color"
	"testing"
)

func TestDownsamplePreservesCoverage(t *testing.T) {
	im := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 2; x++ {
			im.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	got := downsample(im, 4)
	if got.Bounds().Dx() != 1 || got.NRGBAAt(0, 0).R != 255 || got.NRGBAAt(0, 0).A != 127 {
		t.Fatal(got.Bounds(), got.NRGBAAt(0, 0))
	}
}
