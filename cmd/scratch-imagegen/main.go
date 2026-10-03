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
	"strings"
	"talkingduck/menubarimage"
)

func save(path string, im image.Image) error {
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	return png.Encode(f, im)
}
func run() error {
	input := flag.String("input", "", "JSON array of number/stage/state cells (defaults to synthetic style sample)")
	dir := flag.String("out", ".tasks/menubar-images", "output directory")
	agents := flag.Bool("agent-backdrops", false, "preview cropped Claude/Codex/Grok background motifs")
	compact := flag.Bool("compact", false, "use the smaller 4x6 font and tighter spacing")
	flag.Parse()
	mode := menubarimage.Standard
	if *compact {
		mode = menubarimage.Compact
	}
	cells := []menubarimage.Cell{{Number: 1, Stage: "-", State: "idle"}, {Number: 2, Stage: "E", State: "running"}, {Number: 3, Stage: "Q", State: "attention"}, {Number: 4, Stage: "H", State: "unknown"}, {Number: 5, Stage: "d", State: "idle"}, {Number: 6, Stage: "R", State: "running"}, {Number: 7, Stage: "F", State: "attention"}, {Number: 8, Stage: "L", State: "unknown"}, {Number: 9, Stage: "P", State: "idle"}, {Number: 10, Stage: "H", State: "attention"}}
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
		if c.Number < 1 || c.Number > 99 || len(c.Stage) != 1 || !strings.ContainsRune("0123456789HLEFQRPd-?", []rune(c.Stage)[0]) {
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
		icon := menubarimage.Render(cells, theme.fg, mode)
		if *agents {
			icon = agentBackdrop(icon, cells, mode, theme.fg)
		}
		if e := save(filepath.Join(*dir, theme.name+"-icon@2x.png"), menubarimage.Scale(icon, 2)); e != nil {
			return e
		}
		preview := image.NewNRGBA(image.Rect(0, 0, icon.Bounds().Dx()+16, icon.Bounds().Dy()+8))
		draw.Draw(preview, preview.Bounds(), image.NewUniform(theme.bg), image.Point{}, draw.Src)
		draw.Draw(preview, image.Rect(8, 4, 8+icon.Bounds().Dx(), 4+icon.Bounds().Dy()), icon, image.Point{}, draw.Over)
		for _, n := range []int{1, 4} {
			p := filepath.Join(*dir, fmt.Sprintf("%s-preview-%dx.png", theme.name, n))
			if e := save(p, menubarimage.Scale(preview, n)); e != nil {
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
