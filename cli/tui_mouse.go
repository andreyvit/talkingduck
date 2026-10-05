package cli

import (
	"reflect"

	"github.com/gdamore/tcell/v2"
	"talkingduck/table"
)

type tuiMouse struct {
	layout        table.Layout
	width, height int
	x, y          int
	known, down   bool
	pressed       *tuiMouseTarget
}

type tuiMouseTarget struct {
	hit    table.Hit
	choice int // -1 for the table
}

func (s *tuiState) mouseHit(x, y int) (table.Hit, bool) {
	m := &s.mouseState
	if m.width < 24 || m.height < 7 || x < 0 || x >= m.width-1 || y < 2 || y >= m.height-3 {
		return table.Hit{}, false
	}
	hit, ok := m.layout.HitTest(max(0, x-1+s.horizontal), y-2+s.top)
	if x == 0 {
		hit.Source = nil
		hit.CopyControl = false
	}
	return hit, ok
}

func (s *tuiState) mouseTarget(x, y int) *tuiMouseTarget {
	if len(s.choices) > 0 {
		start := max(2, (s.mouseState.height-len(s.choices)-2)/2)
		n := y - start - 1
		if x >= 3 && x < s.mouseState.width-3 && n >= 0 && n < len(s.choices) && y < s.mouseState.height-3 {
			return &tuiMouseTarget{choice: n}
		}
		return nil // The chooser owns input; never click through it.
	}
	hit, ok := s.mouseHit(x, y)
	if !ok {
		return nil
	}
	return &tuiMouseTarget{hit: hit, choice: -1}
}

func (s *tuiState) mouse(e *tcell.EventMouse) {
	m := &s.mouseState
	m.x, m.y = e.Position()
	m.known = true
	buttons := e.Buttons()
	delta := 0
	if buttons&tcell.WheelUp != 0 {
		delta = -3
	}
	if buttons&tcell.WheelDown != 0 {
		delta = 3
	}
	if delta != 0 {
		m.pressed = nil
		if len(s.choices) > 0 {
			s.choice = max(0, min(len(s.choices)-1, s.choice+delta))
		} else {
			s.freeScroll = true
			s.top = max(0, s.top+delta)
		}
	}
	if buttons&tcell.WheelLeft != 0 {
		s.horizontal = max(0, s.horizontal-8)
		m.pressed = nil
	}
	if buttons&tcell.WheelRight != 0 {
		s.horizontal += 8
		m.pressed = nil
	}
	down := buttons&tcell.Button1 != 0
	target := s.mouseTarget(m.x, m.y)
	if down && !m.down {
		m.pressed = target
		if target != nil {
			if target.choice >= 0 {
				s.choice = target.choice
			} else {
				s.selected = target.hit.Row
				s.chord = ""
			}
		}
	}
	if !down && m.down {
		pressed := m.pressed
		m.pressed = nil
		if pressed != nil && reflect.DeepEqual(pressed, target) {
			if target.choice >= 0 {
				item := s.choices[target.choice]
				s.choices = nil
				s.perform(s.verb, item)
			} else if target.hit.Source != nil {
				item := target.hit.Source.Resolve(s.model)
				if target.hit.CopyControl && item.Copy != nil {
					s.perform("c", item)
				} else if !target.hit.CopyControl && item.Link != nil {
					s.perform("o", item)
				}
			}
		}
	}
	m.down = down
}
