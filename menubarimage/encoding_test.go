package menubarimage

import (
	"bytes"
	"image/color"
	"image/png"
	"io"
	"testing"
)

func BenchmarkTrayPNG(b *testing.B) {
	cells := make([]Cell, 10)
	for i := range cells {
		cells[i] = Cell{Number: i + 1, Stage: "HL", State: "running"}
	}
	im := Render(cells, color.NRGBA{255, 255, 255, 255}, Standard)
	for _, level := range []struct {
		name  string
		value png.CompressionLevel
	}{{"default", png.DefaultCompression}, {"fast", png.BestSpeed}} {
		b.Run(level.name, func(b *testing.B) {
			enc := png.Encoder{CompressionLevel: level.value}
			b.ReportAllocs()
			for b.Loop() {
				if err := enc.Encode(io.Discard, im); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
	b.Run("pooled-fast", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if err := EncodePNG(io.Discard, im); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestEncodePNGPixels(t *testing.T) {
	im := Render([]Cell{{Number: 1, Stage: "HL", State: "running"}}, color.NRGBA{255, 255, 255, 255}, Standard)
	for range 2 {
		var buf bytes.Buffer
		if err := EncodePNG(&buf, im); err != nil {
			t.Fatal(err)
		}
		decoded, err := png.Decode(&buf)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Bounds() != im.Bounds() {
			t.Fatal("bounds changed")
		}
		for y := 0; y < im.Bounds().Dy(); y++ {
			for x := 0; x < im.Bounds().Dx(); x++ {
				if color.NRGBAModel.Convert(decoded.At(x, y)) != color.NRGBAModel.Convert(im.At(x, y)) {
					t.Fatalf("pixel changed at %d,%d", x, y)
				}
			}
		}
	}
}
