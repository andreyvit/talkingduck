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
		if im.Bounds().Dx() != 53*4 || im.Bounds().Dy() != 22*4 {
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
	if im.NRGBAAt(0, 32) == track {
		t.Fatal("full not filled")
	}
	if im.NRGBAAt(0, 60) != track {
		t.Fatal("missing was drawn as available")
	}
}
