package worktree

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var agentIndexAssignment = regexp.MustCompile(`^\s*(?:export\s+)?AGENT_INDEX\s*=(.*)$`)
var agentIndexLiteral = regexp.MustCompile(`^(?:([0-9]+)|"([0-9]+)"|'([0-9]+)')(?:\s+#.*)?\s*$`)

// AgentIndex reads a directory-local literal, then a numeric folder suffix.
// It never evaluates shell code or inherits the running app's environment.
// Missing indexes have no value here: the harness's implicit 1 is not assigned.
func AgentIndex(root string) (int, bool) {
	b, err := os.ReadFile(filepath.Join(root, ".env.local"))
	if err != nil && !os.IsNotExist(err) {
		return 0, false
	}
	value, assigned := "", false
	for _, line := range strings.Split(string(b), "\n") {
		if match := agentIndexAssignment.FindStringSubmatch(line); match != nil {
			value, assigned = strings.TrimSpace(match[1]), true
		}
	}
	if assigned {
		match := agentIndexLiteral.FindStringSubmatch(value)
		if match == nil {
			return 0, false
		}
		value = match[1] + match[2] + match[3]
	} else {
		name := filepath.Base(filepath.Clean(root))
		start := len(name)
		for start > 0 && name[start-1] >= '0' && name[start-1] <= '9' {
			start--
		}
		value = name[start:]
	}
	index, err := strconv.Atoi(value)
	return index, err == nil && index > 0
}
