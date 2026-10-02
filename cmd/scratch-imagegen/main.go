// scratch-imagegen renders a two-row menu bar experiment without GUI dependencies.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
)

type Cell struct {
	Number int    `json:"number"`
	Stage  string `json:"stage"`
	State  string `json:"state"` // idle, running, attention, unknown
}

// Deliberately pixel-aligned 5x7 font: one pixel = one logical point.
var glyphs = map[rune][]string{
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

const cellW, cellH, gap = 23, 11, 2

func render(cells []Cell, fg color.NRGBA) *image.NRGBA {
	cols := (len(cells) + 1) / 2
	out := image.NewNRGBA(image.Rect(0, 0, cols*(cellW+gap)-gap, 23))
	dim := fg
	dim.A = 115
	for i, c := range cells {
		x, y := (i%cols)*(cellW+gap), (i/cols)*12
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
		label := strconv.Itoa(c.Number) + c.Stage
		tx := x + (cellW-(len(label)*6-1))/2
		for _, r := range label {
			for yy, row := range glyphs[r] {
				for xx, v := range row {
					if v == '1' {
						out.SetNRGBA(tx+xx, y+2+yy, ink)
					}
				}
			}
			tx += 6
		}
	}
	return out
}
func save(path string, im image.Image) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return png.Encode(f, im)
}
func scale(im image.Image, n int) *image.NRGBA {
	b := im.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()*n, b.Dy()*n))
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			out.Set(x, y, im.At(b.Min.X+x/n, b.Min.Y+y/n))
		}
	}
	return out
}
func run() error {
	input := flag.String("input", "", "JSON array of number/stage/state cells (defaults to synthetic style sample)")
	dir := flag.String("out", ".tasks/menubar-images", "output directory")
	flag.Parse()
	cells := []Cell{{1, "-", "idle"}, {2, "E", "running"}, {3, "Q", "attention"}, {4, "H", "unknown"}, {5, "d", "idle"}, {6, "R", "running"}, {7, "F", "attention"}, {8, "L", "unknown"}, {9, "P", "idle"}, {10, "H", "attention"}}
	if *input != "" {
		b, e := os.ReadFile(*input)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(b, &cells); e != nil {
			return e
		}
	}
	if len(cells) == 0 || len(cells) > 20 {
		return fmt.Errorf("expected 1–20 cells")
	}
	for _, c := range cells {
		if c.Number < 1 || c.Number > 99 || len(c.Stage) != 1 || glyphs[[]rune(c.Stage)[0]] == nil {
			return fmt.Errorf("invalid cell: %+v", c)
		}
		switch c.State {
		case "idle", "running", "attention", "unknown":
		default:
			return fmt.Errorf("invalid state %q", c.State)
		}
	}
	if e := os.MkdirAll(*dir, 0755); e != nil {
		return e
	}
	for _, theme := range []struct {
		name   string
		fg, bg color.NRGBA
	}{{"dark", color.NRGBA{240, 242, 246, 255}, color.NRGBA{34, 37, 43, 255}}, {"light", color.NRGBA{30, 33, 39, 255}, color.NRGBA{233, 236, 242, 255}}} {
		icon := render(cells, theme.fg)
		if e := save(filepath.Join(*dir, theme.name+"-icon@2x.png"), scale(icon, 2)); e != nil {
			return e
		}
		preview := image.NewNRGBA(image.Rect(0, 0, icon.Bounds().Dx()+16, 31))
		draw.Draw(preview, preview.Bounds(), image.NewUniform(theme.bg), image.Point{}, draw.Src)
		draw.Draw(preview, image.Rect(8, 4, 8+icon.Bounds().Dx(), 27), icon, image.Point{}, draw.Over)
		for _, n := range []int{1, 4} {
			p := filepath.Join(*dir, fmt.Sprintf("%s-preview-%dx.png", theme.name, n))
			if e := save(p, scale(preview, n)); e != nil {
				return e
			}
			fmt.Println(p)
		}
	}
	return nil
}
func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
