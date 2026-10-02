package menubarimage

import (
	"image/color"
	"testing"
)

func TestAttentionIsTransparentCutout(t *testing.T) {
	for _, tc := range []struct {
		mode Mode
		x    int
	}{{Standard, 8}, {Compact, 6}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			fg := color.NRGBA{A: 255}
			im := Render([]Cell{{Number: 1, Stage: "L", State: "attention"}}, fg, tc.mode)
			if got := im.NRGBAAt(0, 0); got != fg {
				t.Fatalf("fill=%v", got)
			}
			if got := im.NRGBAAt(tc.x, 2); got.A != 0 {
				t.Fatalf("letter must be transparent, got %v", got)
			}
		})
	}
}

func TestGridAndRetinaDimensions(t *testing.T) {
	cells := make([]Cell, 10)
	for i := range cells {
		cells[i] = Cell{Number: i + 1, Stage: "H", State: "running"}
	}
	for _, tc := range []struct {
		mode Mode
		w, h int
	}{{Standard, 123, 23}, {Compact, 99, 21}} {
		t.Run(string(tc.mode), func(t *testing.T) {
			im := Render(cells, color.NRGBA{A: 255}, tc.mode)
			if im.Bounds().Dx() != tc.w || im.Bounds().Dy() != tc.h {
				t.Fatal(im.Bounds())
			}
			retina := Scale(im, 2)
			if retina.Bounds().Dx() != tc.w*2 || retina.Bounds().Dy() != tc.h*2 {
				t.Fatal(retina.Bounds())
			}
			for y := 0; y < tc.h*2; y++ {
				for x := 0; x < tc.w*2; x++ {
					if retina.NRGBAAt(x, y) != im.NRGBAAt(x/2, y/2) {
						t.Fatalf("scale changed pixel %d,%d", x, y)
					}
				}
			}
		})
	}
}
