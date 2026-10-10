package settings

import "fmt"

type MarkdownAppearance struct {
	Font string `json:"font"`
	Size int    `json:"size"`
}

func (a MarkdownAppearance) Defaults() MarkdownAppearance {
	if a.Font == "" {
		a.Font = "github"
	}
	if a.Size == 0 {
		a.Size = 16
	}
	return a
}
func (s *Store) SetMarkdownAppearance(a MarkdownAppearance) (Data, error) {
	a = a.Defaults()
	if len(a.Font) > 200 || a.Size < 10 || a.Size > 40 {
		return Data{}, fmt.Errorf("Markdown font size must be 10–40 px")
	}
	return s.update(func(d *Data) error { d.MarkdownAppearance = a; return nil })
}
