package menubarimage

import (
	"image"
	"image/png"
	"io"
	"sync"
)

type pngBuffers struct{ sync.Pool }

func (p *pngBuffers) Get() *png.EncoderBuffer {
	b, _ := p.Pool.Get().(*png.EncoderBuffer)
	return b
}
func (p *pngBuffers) Put(b *png.EncoderBuffer) { p.Pool.Put(b) }

var trayEncoder = png.Encoder{CompressionLevel: png.BestSpeed, BufferPool: &pngBuffers{}}

// EncodePNG reuses compression buffers across frequent menu-bar updates.
func EncodePNG(w io.Writer, im image.Image) error { return trayEncoder.Encode(w, im) }
