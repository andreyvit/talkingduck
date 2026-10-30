package worktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentIndex(t *testing.T) {
	for _, tt := range []struct {
		name, folder, env string
		want              int
		ok                bool
	}{
		{"absent", "feature", "", 0, false},
		{"suffix", "back12", "", 12, true},
		{"explicit one", "feature", "export AGENT_INDEX=1\n", 1, true},
		{"override suffix", "back12", "AGENT_INDEX=7\n", 7, true},
		{"double quote", "feature", " export AGENT_INDEX = \"19\" # assigned\r\n", 19, true},
		{"single quote", "feature", "AGENT_INDEX='11'\n", 11, true},
		{"last assignment", "feature", "AGENT_INDEX=3\nAGENT_INDEX=14\n", 14, true},
		{"comment only", "feature", "# AGENT_INDEX=4\n", 0, false},
		{"invalid override", "back12", "AGENT_INDEX=bad\n", 0, false},
		{"shell expression", "back12", "AGENT_INDEX=$(echo 7)\n", 0, false},
		{"empty override", "back12", "AGENT_INDEX=\n", 0, false},
		{"zero", "feature", "AGENT_INDEX=0\n", 0, false},
		{"outside shortcut range", "feature", "AGENT_INDEX=20\n", 20, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), tt.folder)
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			if tt.env != "" {
				if err := os.WriteFile(filepath.Join(root, ".env.local"), []byte(tt.env), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("AGENT_INDEX", "9") // Never inherit the app launcher's index.
			got, ok := AgentIndex(root)
			if ok != tt.ok || ok && got != tt.want {
				t.Fatalf("got %d,%v; want %d,%v", got, ok, tt.want, tt.ok)
			}
		})
	}
}
