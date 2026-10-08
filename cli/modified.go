package cli

import (
	"strconv"
	"strings"
)

// Keep non-numeric states visible instead of converting unknown counts to zero.
func modifiedCount(tracked, untracked string) string {
	if strings.TrimSpace(tracked) == "" && strings.TrimSpace(untracked) == "" {
		return ""
	}
	if tracked == "ERROR" {
		return tracked
	}
	a, e1 := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(tracked, "±")))
	b, e2 := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(untracked, "?")))
	if e1 != nil || e2 != nil {
		return "±?"
	}
	return "±" + strconv.Itoa(a+b)
}
