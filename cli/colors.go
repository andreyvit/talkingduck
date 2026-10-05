package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const gray = 242

func useColor(mode string, out io.Writer) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default:
		f, ok := out.(*os.File)
		return ok && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb" && term.IsTerminal(int(f.Fd()))
	}
}

func paint(text string, color int) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	body := strings.TrimRight(text, " ")
	return fmt.Sprintf("\x1b[38;5;%dm%s\x1b[0m%s", color, body, text[len(body):])
}

func paintNumber(text, prefix string, color int) string {
	n := strings.TrimSpace(strings.TrimPrefix(text, prefix))
	if n == "0" || n == "?" || n == "--" {
		color = gray
	}
	return paint(text, color)
}

// Colors are applied after measuring, wrapping, and padding plain terminal cells.
func colorCell(text string, column int) string {
	value := strings.TrimSpace(text)
	if value == "" {
		return text
	}
	if column == 14 {
		return paint(text, 15)
	} // Keep the trailing title white.
	if column == 1 {
		color := map[string]int{"Codex": 81, "Claude": 209, "Grok": 141}[strings.TrimSuffix(value, "!")]
		if color == 0 {
			color = gray
		}
		if strings.HasSuffix(value, "!") {
			i := strings.Index(text, "!")
			return paint(text[:i], color) + paint(text[i:i+1], 203) + text[i+1:]
		}
		return paint(text, color)
	}
	if column == 4 {
		if value == "CLEAN" || value == "DIRTY" || value == "MARKDN" {
			return paint(text, 15)
		}
		color := map[string]int{"INV": 75, "HL": 75, "LL": 141, "EX": 81, "FIN": 114, "SQ": 212, "PUSH": 221, "PUSHED": 245, "REBA": 209}[value]
		if color == 0 {
			color = gray
		}
		return paint(text, color)
	}
	if column == 6 || column == 7 {
		color := map[string]int{"▶": 81, "✓": 114, "!": 221}[value]
		if color == 0 {
			color = gray
		}
		return paint(text, color)
	}
	if column == 3 {
		return paintNumber(text, "", 141)
	}
	if column == 5 || column == 8 {
		if parts := strings.SplitN(text, "/", 2); len(parts) == 2 {
			left, right := 114, 75
			if column == 8 {
				left, right = 221, 209
			}
			return paintNumber(parts[0], "", left) + paint("/", gray) + paintNumber(parts[1], "", right)
		}
	}
	if column >= 9 && column <= 13 {
		prefixes := []string{"±", "?", "ℹ︎", "⇣", "⇡"}
		colors := []int{221, 114, 75, 203, 81}
		if value == "ERROR" {
			return paint(text, 203)
		}
		return paintNumber(text, prefixes[column-9], colors[column-9])
	}
	if value == "?" || value == "--" {
		return paint(text, gray)
	}
	return text
}
