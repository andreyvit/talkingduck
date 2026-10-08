package cli

import (
	"strconv"
	"strings"
)

// Keep non-numeric states visible instead of converting unknown counts to zero.
func modifiedCount(tracked, untracked, markdown string) string {
	if strings.TrimSpace(tracked) == "" && strings.TrimSpace(untracked) == "" && strings.TrimSpace(markdown) == "" {
		return ""
	}
	if tracked == "ERROR" {
		return tracked
	}
	a, e1 := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(tracked, "±")))
	b, e2 := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(untracked, "?")))
	c, e3 := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(markdown, "ℹ︎")))
	if e1 != nil || e2 != nil || e3 != nil {
		return "±?"
	}
	if a+b == 0 && c > 0 {
		return ".md"
	}
	return "±" + strconv.Itoa(a+b+c)
}
