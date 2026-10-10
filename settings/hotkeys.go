package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// Nil preserves the original default; all empty entries explicitly disable it.
type Hotkeys [3]string

func (d Data) EffectiveHotkeys() Hotkeys {
	if d.GlobalHotkeys == nil {
		return Hotkeys{"Ctrl+T", ""}
	}
	return *d.GlobalHotkeys
}
func ValidateHotkeys(keys Hotkeys) error {
	for _, value := range keys {
		if value == "" {
			continue
		}
		parts := strings.Split(value, "+")
		seen := map[string]bool{}
		for _, mod := range parts[:len(parts)-1] {
			if !strings.Contains("|Ctrl|Alt|Shift|Super|", "|"+mod+"|") || seen[mod] {
				return fmt.Errorf("invalid hotkey modifier: %s", mod)
			}
			seen[mod] = true
		}
		key := parts[len(parts)-1]
		valid := len(key) == 1 && strings.Contains("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789=-[]';\\,/.`", key)
		valid = valid || strings.Contains("|Return|Tab|Space|Backspace|Delete|Escape|Home|Page Up|End|Page Down|Left|Right|Down|Up|", "|"+key+"|")
		if strings.HasPrefix(key, "F") {
			n, e := strconv.Atoi(key[1:])
			valid = valid || e == nil && n >= 1 && n <= 20
		}
		if !valid || key == "" {
			return fmt.Errorf("unsupported hotkey key: %s", key)
		}
	}
	// Compare modifier sets, not spelling order.
	canonical := func(s string) string {
		parts := strings.Split(s, "+")
		result := ""
		for _, m := range []string{"Ctrl", "Alt", "Shift", "Super"} {
			for _, p := range parts[:len(parts)-1] {
				if p == m {
					result += m + "+"
				}
			}
		}
		return result + parts[len(parts)-1]
	}
	seenKeys := map[string]bool{}
	for _, key := range keys {
		if key == "" {
			continue
		}
		c := canonical(key)
		if seenKeys[c] {
			return fmt.Errorf("choose different hotkeys")
		}
		seenKeys[c] = true
	}
	return nil
}
func (s *Store) SetHotkeys(keys Hotkeys) (Data, error) {
	if err := ValidateHotkeys(keys); err != nil {
		return Data{}, err
	}
	return s.update(func(d *Data) error { d.GlobalHotkeys = &keys; return nil })
}
