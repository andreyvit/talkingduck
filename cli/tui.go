package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
	"talkingduck/actions"
	"talkingduck/inventory"
	"talkingduck/monitor"
	"talkingduck/table"
)

type tuiState struct {
	model                     table.Table
	mouseState                tuiMouse
	freeScroll                bool
	selected, top, horizontal int
	chord, notice, verb       string
	choices                   []table.Item
	choice                    int
	refresh                   func(string)
	perform                   func(string, table.Item)
}

func (s *tuiState) replace(model table.Table) {
	id := ""
	if s.selected < len(s.model.Rows) {
		id = s.model.Rows[s.selected].ID
	}
	s.mouseState.pressed = nil
	s.model = model
	s.selected = min(s.selected, max(0, len(model.Rows)-1))
	for n, r := range model.Rows {
		if r.ID == id {
			s.selected = n
			break
		}
	}
	// Keep a pending chord through unrelated updates, but never act on stale targets.
	if len(model.Rows) == 0 || model.Rows[s.selected].ID != id {
		s.choices = nil
		s.chord = ""
	} else if len(s.choices) > 0 {
		current := model.Rows[s.selected].Matches(s.verb, s.choices[0].Chord)
		if !reflect.DeepEqual(current, s.choices) {
			s.choices = nil
			s.chord = ""
		}
	}
}
func (s *tuiState) key(e *tcell.EventKey) bool {
	if e.Key() == tcell.KeyCtrlC || (e.Key() == tcell.KeyRune && e.Rune() == 'Q') {
		return true
	}
	if e.Key() == tcell.KeyEscape {
		s.chord = ""
		s.choices = nil
		return false
	}
	if len(s.choices) > 0 {
		switch e.Key() {
		case tcell.KeyUp:
			s.choice = max(0, s.choice-1)
		case tcell.KeyDown:
			s.choice = min(len(s.choices)-1, s.choice+1)
		case tcell.KeyEnter:
			item := s.choices[s.choice]
			s.choices = nil
			s.perform(s.verb, item)
		}
		return false
	}
	switch e.Key() {
	case tcell.KeyUp:
		s.freeScroll = false
		s.selected = max(0, s.selected-1)
		s.chord = ""
		return false
	case tcell.KeyDown:
		s.freeScroll = false
		s.selected = min(max(0, len(s.model.Rows)-1), s.selected+1)
		s.chord = ""
		return false
	case tcell.KeyLeft:
		s.horizontal = max(0, s.horizontal-8)
		return false
	case tcell.KeyRight:
		s.horizontal += 8
		return false
	}
	if e.Key() != tcell.KeyRune {
		return false
	}
	r := e.Rune()
	if s.chord == "" {
		switch r {
		case 'r':
			if len(s.model.Rows) > 0 {
				s.refresh(s.model.Rows[s.selected].Group)
				s.notice = "Refresh queued"
			}
		case 'R':
			s.refresh("")
			s.notice = "Refresh all queued"
		case 'o', 'c':
			s.chord = string(r)
		}
		return false
	}
	if len(s.model.Rows) == 0 {
		s.chord = ""
		return false
	}
	s.chord += string(r)
	verb, key := s.chord[:1], s.chord[1:]
	row := s.model.Rows[s.selected]
	items := row.Matches(verb, key)
	if len(items) > 0 {
		s.chord = ""
		if len(items) == 1 {
			s.perform(verb, items[0])
		} else {
			s.choices = items
			s.choice = 0
			s.verb = verb
		}
		return false
	}
	for _, candidate := range row.Chords(verb) {
		if strings.HasPrefix(candidate, key) {
			return false
		}
	}
	s.notice = "Action unavailable for this row"
	if key == "g" {
		s.notice = "Git opening is not implemented yet"
	}
	s.chord = ""
	return false
}

var tuiColors = map[string]string{"dim": "#777f90", "muted": "#81899a", "cyan": "#62cce9", "orange": "#f4a172", "purple": "#ba96df", "green": "#9cce85", "yellow": "#e5c66b", "blue": "#77b8ec", "red": "#f18181"}

func tuiColor(name string) tcell.Color {
	if hex, ok := tuiColors[name]; ok {
		return tcell.GetColor(hex)
	}
	return tcell.GetColor("#e7e8ec")
}
func put(screen tcell.Screen, x, y, limit int, text string, style tcell.Style) {
	for text != "" && x < limit {
		before := text
		rest, w := screen.Put(x, y, text, style)

		if rest == before {
			return
		}
		x += w
		text = rest
	}
}
func (s *tuiState) draw(screen tcell.Screen) {
	w, h := screen.Size()
	s.mouseState.width, s.mouseState.height = w, h
	s.mouseState.layout = table.Layout{}
	base := tcell.StyleDefault.Background(tcell.GetColor("#191e29")).Foreground(tuiColor(""))
	screen.SetStyle(base)
	screen.Clear()
	if w < 24 || h < 7 {
		put(screen, 0, 0, w, "Enlarge terminal · Q quit", base)
		screen.Show()
		return
	}
	border := base.Foreground(tuiColor("blue"))
	bar := base.Background(tcell.GetColor("#20394e")).Foreground(tuiColor("cyan")).Bold(true)
	for x := 0; x < w; x++ {
		screen.SetContent(x, 0, '─', nil, border)
		screen.SetContent(x, h-3, '─', nil, border)
	}
	put(screen, 2, 0, w-1, " TalkingDuck ", bar)
	display := s.model
	display.Columns = append([]table.Column(nil), s.model.Columns...)
	for i := range display.Columns {
		display.Columns[i].MinWidth = max(display.Columns[i].MinWidth, table.Width(display.Columns[i].Label))
	}
	layout := table.FormatWithOptions(display, w-2, table.FormatOptions{CopyControl: "©"})
	s.mouseState.layout = layout
	total := max(0, len(layout.Widths)-1)
	for _, width := range layout.Widths {
		total += width
	}
	s.horizontal = min(s.horizontal, max(0, total-(w-2)))
	x := 1 - s.horizontal
	for n, c := range s.model.Columns {
		label := c.Label
		if table.Width(label) > layout.Widths[n] {
			label = string([]rune(label)[:min(len([]rune(label)), layout.Widths[n])])
		}
		put(screen, x, 1, w-1, label, bar)
		x += layout.Widths[n] + 1
	}
	viewport := h - 5
	first, last := -1, -1
	for n, l := range layout.Lines {
		if l.Row == s.selected {
			if first < 0 {
				first = n
			}
			last = n
		}
	}
	if first >= 0 && !s.freeScroll {
		if first < s.top {
			s.top = first
		}
		if last >= s.top+viewport {
			s.top = max(first, last-viewport+1)
		}
	}
	s.top = max(0, min(s.top, max(0, len(layout.Lines)-viewport)))
	hover, _ := s.mouseHit(s.mouseState.x, s.mouseState.y)
	for line := s.top; line < len(layout.Lines) && line < s.top+viewport; line++ {
		visual := layout.Lines[line]
		row := s.model.Rows[visual.Row]
		style := base
		switch row.State {
		case "running":
			style = style.Background(tcell.GetColor("#20363e"))
		case "attention":
			style = style.Background(tcell.GetColor("#363126"))
		}
		if visual.Row == s.selected {
			style = style.Background(tcell.GetColor("#254862"))
		}
		y := 2 + line - s.top
		for xx := 0; xx < w; xx++ {
			screen.SetContent(xx, y, ' ', nil, style)
		}
		if visual.Row == s.selected {
			screen.SetContent(0, y, '▸', nil, style.Foreground(tuiColor("cyan")))
		}
		for _, span := range visual.Spans {
			if span.CopyControl && (len(s.choices) > 0 || !s.mouseState.known || hover.Source == nil || span.Source == nil || *hover.Source != *span.Source) {
				continue
			}
			put(screen, 1+span.X-s.horizontal, y, w-1, span.Text, style.Foreground(tuiColor(span.Color)).Bold(span.Bold))
		}
	}
	message := s.notice
	if len(s.model.Rows) > 0 {
		message = s.model.Rows[s.selected].Group + " · " + message
	}
	if s.chord != "" {
		message = s.chord + "…  a agent  t ticket  s spec  f folder  g git (soon) · Esc cancel"
	}
	put(screen, 1, h-2, w-1, message, base.Foreground(tuiColor("cyan")))
	put(screen, 1, h-1, w-1, "↑↓ select  ←→ scroll  r refresh  R all  o open  c copy  Q quit", bar)
	if len(s.choices) > 0 {
		y := max(2, (h-len(s.choices)-2)/2)
		put(screen, 3, y, w-3, "Choose item · ↑↓ Enter · Esc cancel", bar)
		for n, item := range s.choices {
			if y+n+1 >= h-3 {
				break
			}
			style := base
			if n == s.choice {
				style = bar
			}
			put(screen, 3, y+n+1, w-3, item.Text, style)
		}
	}
	screen.Show()
}

// RunTUI uses the same cached inventory and queue policies as the desktop app.
// Full Git scans never run on the terminal's input/drawing goroutine.
func RunTUI(opts *Options, paths []string) int {
	if opts.Inventory.AgentLimit < 0 {
		fmt.Fprintln(os.Stderr, "-agent-limit must be nonnegative")
		return 2
	}
	config := opts.Inventory
	if len(paths) > 0 {
		config.Paths = paths
	}
	cache, err := inventory.New(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err = screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer screen.Fini()
	screen.EnableMouse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	events := make(chan tcell.Event, 16)
	go screen.ChannelEvents(events, ctx.Done())
	updates := make(chan struct{}, 1)
	notices := make(chan string, 16)
	publish := func() {
		select {
		case updates <- struct{}{}:
		default:
		}
	}
	tell := func(msg string) {
		select {
		case notices <- msg:
		default:
		}
	}
	cheap, full := monitor.NewQueue(monitor.Cheap), monitor.NewQueue(monitor.Expensive)
	probe := inventory.NewChangeProbe(cache.Paths(), "", !config.NoAgents)
	var emit func(inventory.ChangeEvent)
	emit = func(e inventory.ChangeEvent) {
		if e.Reconcile {
			cheap.Notify("@providers")
			for _, p := range cache.Paths() {
				cheap.Notify(p)
				full.Notify(p)
			}
			return
		}
		if e.Providers {
			cheap.Notify("@providers")
		}
		if e.Path != "" {
			if e.Cheap {
				cheap.Notify(e.Path)
			}
			if e.Expensive {
				full.Notify(e.Path)
			}
		}
	}
	go cheap.Run(ctx, func(ctx context.Context, path string) error {
		if path == "@providers" {
			changed, warnings := probe.Poll()
			for _, w := range warnings {
				tell(w)
			}
			for _, p := range changed {
				emit(inventory.ChangeEvent{Path: p, Cheap: true, Expensive: true})
			}
		} else {
			_, err := cache.UpdateLive(path)
			publish()
			return err
		}
		return nil
	})
	go full.Run(ctx, func(ctx context.Context, path string) error {
		tell("Refreshing " + path)
		err := cache.Update(path)
		if err != nil {
			tell(err.Error())
		} else {
			tell("Updated " + path)
		}
		publish()
		return err
	})
	go inventory.WatchChanges(ctx, cache.Paths(), "", !config.NoAgents, 10*time.Minute, emit, func(err error) { tell(err.Error()) })
	if !config.NoAgents {
		go inventory.NewProcessChanges(cache.Paths(), "").Run(ctx, emit)
	}
	state := tuiState{notice: "Collecting inventory…"}
	state.refresh = func(path string) {
		if path == "" {
			for _, p := range cache.Paths() {
				full.Notify(p)
			}
		} else {
			full.Notify(path)
		}
	}
	state.perform = func(verb string, item table.Item) {
		go func() {
			var err error
			if verb == "o" && item.Link != nil {
				err = actions.Open(*item.Link)
			} else if verb == "c" && item.Copy != nil {
				err = actions.Copy(*item.Copy)
			}
			if err != nil {
				tell(err.Error())
			} else {
				tell("Done: " + item.Text)
			}
		}()
	}
	rebuild := func() {
		view, err := BuildTable(cache.Snapshot().Overview)
		if err != nil {
			state.notice = err.Error()
		} else {
			state.replace(view.Model)
			if len(view.Warnings) > 0 {
				state.notice = view.Warnings[0]
			}
		}
	}
	rebuild()
	for {
		state.draw(screen)
		select {
		case e, ok := <-events:
			if !ok {
				return 0
			}
			switch ev := e.(type) {
			case *tcell.EventKey:
				if state.key(ev) {
					return 0
				}
			case *tcell.EventMouse:
				state.mouse(ev)
			case *tcell.EventResize:
				state.mouseState.pressed = nil
				screen.Sync()
			}
		case <-updates:
			rebuild()
		case msg := <-notices:
			state.notice = msg
		case <-ctx.Done():
			return 0
		}
	}
}
