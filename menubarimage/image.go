// Package menubarimage draws compact, pixel-aligned checkout status grids.
package menubarimage

import (
	"image"
	"image/color"
	"image/draw"
	"strconv"
)

type Cell struct {
	Shortcut string `json:"shortcut,omitempty"`
	Agent    string `json:"agent,omitempty"`
	Number   int    `json:"number"`
	Stage    string `json:"stage"`
	State    string `json:"state"` // idle, running, attention, unknown
}

// Deliberately pixel-aligned 4x6 font: one pixel = one logical point.
var compactGlyphs = map[rune][]string{
	'0': {"0110", "1001", "1001", "1001", "1001", "0110"},
	'1': {"0100", "1100", "0100", "0100", "0100", "1110"},
	'2': {"1110", "0001", "0010", "0100", "1000", "1111"},
	'3': {"1110", "0001", "0110", "0001", "0001", "1110"},
	'4': {"1001", "1001", "1111", "0001", "0001", "0001"},
	'5': {"1111", "1000", "1110", "0001", "0001", "1110"},
	'6': {"0111", "1000", "1110", "1001", "1001", "0110"},
	'7': {"1111", "0001", "0010", "0100", "0100", "0100"},
	'8': {"0110", "1001", "0110", "1001", "1001", "0110"},
	'9': {"0110", "1001", "0111", "0001", "0001", "1110"},
	'H': {"1001", "1001", "1111", "1001", "1001", "1001"},
	'L': {"1000", "1000", "1000", "1000", "1000", "1111"},
	'E': {"1111", "1000", "1110", "1000", "1000", "1111"},
	'F': {"1111", "1000", "1110", "1000", "1000", "1000"},
	'Q': {"0110", "1001", "1001", "1001", "1011", "0111"},
	'R': {"1110", "1001", "1110", "1100", "1010", "1001"},
	'P': {"1110", "1001", "1110", "1000", "1000", "1000"},
	'd': {"0001", "0001", "0111", "1001", "1001", "0111"},
	'-': {"0000", "0000", "1111", "0000", "0000", "0000"},
	'?': {"0110", "1001", "0010", "0100", "0000", "0100"},
}

var standardGlyphs = map[rune][]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'd': {"00001", "00001", "01101", "10011", "10001", "10001", "01111"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'?': {"01110", "10001", "00001", "00010", "00100", "00000", "00100"},
}

type Mode string

const (
	Standard Mode = "standard"
	Compact  Mode = "compact"
)

// Render draws two rows in input order. Empty input produces an empty image.
func Render(cells []Cell, fg color.NRGBA, mode Mode) *image.NRGBA {
	if len(cells) == 0 {
		return image.NewNRGBA(image.Rectangle{})
	}
	cellW, cellH, gap, advance := 23, 11, 2, 6
	rowGap := 1
	glyphs := standardGlyphs
	if mode == Compact {
		cellW, cellH, gap, advance = 19, 10, 1, 5
		glyphs = compactGlyphs
	}
	cols := (len(cells) + 1) / 2
	out := image.NewNRGBA(image.Rect(0, 0, cols*(cellW+gap)-gap, cellH*2+rowGap))
	dim := fg
	dim.A = 115
	for i, c := range cells {
		x, y := (i%cols)*(cellW+gap), (i/cols)*(cellH+rowGap)
		ink := fg
		switch c.State {
		case "attention":
			draw.Draw(out, image.Rect(x, y, x+cellW, y+cellH), image.NewUniform(fg), image.Point{}, draw.Src)
			ink = color.NRGBA{} // transparent lettering reverses against the menu bar
		case "idle", "running":
			border := fg
			if c.State == "idle" {
				border = dim
				ink = dim
			}
			for xx := 0; xx < cellW; xx++ {
				if c.State == "running" || xx%2 == 0 {
					out.SetNRGBA(x+xx, y, border)
					out.SetNRGBA(x+xx, y+cellH-1, border)
				}
			}
			for yy := 0; yy < cellH; yy++ {
				if c.State == "running" || yy%2 == 0 {
					out.SetNRGBA(x, y+yy, border)
					out.SetNRGBA(x+cellW-1, y+yy, border)
				}
			}
		case "unknown":
			// Corner marks distinguish unverified activity from known idle.
			for _, dx := range []int{0, cellW - 1} {
				for _, dy := range []int{0, cellH - 1} {
					out.SetNRGBA(x+dx, y+dy, dim)
				}
			}
		}
		label := c.shortcut() + c.Stage
		tx := x + (cellW-(len(label)*advance-1))/2
		for _, r := range label {
			for yy, row := range glyphs[r] {
				for xx, v := range row {
					if v == '1' {
						out.SetNRGBA(tx+xx, y+2+yy, ink)
					}
				}
			}
			tx += advance
		}
	}
	return out
}

// Scale enlarges without smoothing; n must be positive.
func Scale(im image.Image, n int) *image.NRGBA {
	if n < 1 {
		panic("menubarimage: scale must be positive")
	}
	b := im.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*n, b.Dy()*n))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, im.At(b.Min.X+x/n, b.Min.Y+y/n))
		}
	}
	return out
}

func (c Cell) shortcut() string {
	if c.Shortcut != "" {
		return c.Shortcut
	}
	return strconv.Itoa(c.Number)
}
