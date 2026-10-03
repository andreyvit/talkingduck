package main

import (
	"image"
	"image/color"
	"talkingduck/menubarimage"
)

func agentBackdrop(_ *image.NRGBA, cells []menubarimage.Cell, mode menubarimage.Mode, fg color.NRGBA) *image.NRGBA {
	cells = append([]menubarimage.Cell(nil), cells...)
	for i := range cells {
		if cells[i].Agent == "" {
			cells[i].Agent = []string{"claude", "codex", "grok"}[i%3]
		}
	}
	return menubarimage.RenderAgents(cells, fg, mode)
}
