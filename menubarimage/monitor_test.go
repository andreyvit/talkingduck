package menubarimage

import (
	"image/color"
	"testing"
)

func TestMonitorCompactBounds(t *testing.T) {
	p := 72.0
	for _, n := range []int{1, 2, 3} {
		bars := make([]LimitBar, n)
		for i := range bars {
			bars[i] = LimitBar{ID: "codex", Remaining: &p}
		}
		im := RenderMonitor(Monitor{Ping: "20ms", Bars: bars})
		if im.Bounds().Dx() != 37*4 || im.Bounds().Dy() != 23*4 {
			t.Fatal(im.Bounds())
		}
	}
	if !RenderMonitor(Monitor{}).Bounds().Empty() {
		t.Fatal("empty monitor")
	}
	if RenderMonitor(Monitor{Ping: "3s", PingFailed: true}).Bounds().Dx() != 32 {
		t.Fatal("ping width")
	}
}
func TestMonitorRemainingAndMissing(t *testing.T) {
	zero, full := 0.0, 100.0
	im := RenderMonitor(Monitor{Bars: []LimitBar{{Remaining: &zero}, {Remaining: &full}, {}}})
	track := color.NRGBA{69, 77, 89, 255}
	if im.NRGBAAt(0, 4) != track {
		t.Fatal("zero was drawn as available")
	}
	if im.NRGBAAt(4, 36) == track {
		t.Fatal("full not filled")
	}
	if im.NRGBAAt(4, 68) != track {
		t.Fatal("missing was drawn as available")
	}
}

func TestMonitorGridEdges(t *testing.T) {
	for _, mode := range []Mode{Standard, Compact} {
		im := RenderMonitor(Monitor{Mode: mode, Bars: []LimitBar{{}, {}, {}}})
		projects := RenderAgents([]Cell{{Number: 1}}, color.NRGBA{A: 255}, mode)
		if im.Bounds().Dy() != projects.Bounds().Dy() {
			t.Fatal("grid heights differ", mode)
		}
		if im.NRGBAAt(10, 0).A != 255 || im.NRGBAAt(10, im.Bounds().Dy()-1).A != 255 {
			t.Fatal("outer edges inset", mode)
		}
	}
}
func TestPingColorsAndTopAnchor(t *testing.T) {
	for _, tc := range []struct {
		ms   float64
		want color.NRGBA
	}{{49, color.NRGBA{255, 255, 255, 255}}, {50, color.NRGBA{255, 244, 174, 255}}, {149, color.NRGBA{255, 244, 174, 255}}, {150, color.NRGBA{255, 174, 174, 255}}} {
		im := RenderMonitor(Monitor{Ping: "99 ms", PingMS: tc.ms})
		found := false
		for y := 0; y < im.Bounds().Dy(); y++ {
			for x := 0; x < im.Bounds().Dx(); x++ {
				if im.NRGBAAt(x, y) == tc.want {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("missing ping color", tc.ms)
		}
	}
	a, b := RenderMonitor(Monitor{Ping: "99 ms"}), RenderMonitor(Monitor{Ping: "100 ms"})
	for y := 0; y < 20; y++ {
		for x := 0; x < a.Bounds().Dx(); x++ {
			if a.NRGBAAt(x, y) != b.NRGBAAt(x, y) {
				t.Fatal("unit moved when digit count changed", x, y)
			}
		}
	}
}
