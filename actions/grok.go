package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const cmuxCLI = "/Applications/cmux.app/Contents/Resources/bin/cmux"

type grokHook struct {
	SessionID string  `json:"sessionId"`
	PID       int     `json:"pid"`
	Start     int64   `json:"pidStartSeconds"`
	Updated   float64 `json:"updatedAt"`
	Surface   string  `json:"surfaceId"`
}

func currentGrokHook(records map[string]grokHook, id string) (grokHook, error) {
	h, ok := records[id]
	if !ok || h.SessionID != id || h.PID <= 0 || h.Start <= 0 || h.Surface == "" {
		return h, fmt.Errorf("Session is no longer open (cannot confirm its cmux terminal)")
	}
	for key, other := range records {
		if key != id && (other.Surface == h.Surface || (other.PID == h.PID && other.Start == h.Start)) && other.Updated >= h.Updated {
			return h, fmt.Errorf("Session is no longer open (terminal now belongs to another session)")
		}
	}
	return h, nil
}

func OpenGrok(id string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(home, ".cmuxterm", "grok-hook-sessions.json"))
	if err != nil {
		return fmt.Errorf("Session is no longer open (cmux session records unavailable)")
	}
	var store struct {
		Sessions map[string]grokHook `json:"sessions"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return fmt.Errorf("Cannot read cmux session records: %w", err)
	}
	h, err := currentGrokHook(store.Sessions, id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Verify the process generation, not just a potentially reused PID.
	cmd := exec.CommandContext(ctx, "/bin/ps", "-p", strconv.Itoa(h.PID), "-o", "lstart=", "-o", "comm=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	fields := strings.Fields(string(out))
	if err != nil || len(fields) < 6 {
		return fmt.Errorf("Session is no longer open")
	}
	started, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(fields[:5], " "), time.Local)
	if err != nil || started.Unix() != h.Start || !strings.Contains(strings.ToLower(filepath.Base(fields[5])), "grok") {
		return fmt.Errorf("Session is no longer open (process identity changed)")
	}
	rpc := func(method string, params any, result any) error {
		payload, err := json.Marshal(params)
		if err != nil {
			return err
		}
		out, err := exec.CommandContext(ctx, cmuxCLI, "--json", "rpc", method, string(payload)).Output()
		if err != nil {
			return fmt.Errorf("Cannot reach cmux or confirm its terminal: %w", err)
		}
		if result != nil {
			return json.Unmarshal(out, result)
		}
		return nil
	}
	var target struct {
		Source    string `json:"source"`
		Workspace string `json:"workspace_id"`
		Surface   string `json:"surface_id"`
	}
	if err := rpc("agent.resolve_delivery_target", map[string]int{"pid": h.PID}, &target); err != nil {
		return err
	}
	if target.Source != "pid" || target.Workspace == "" || !strings.EqualFold(target.Surface, h.Surface) {
		return fmt.Errorf("Session is no longer open (terminal association changed)")
	}
	if err := rpc("workspace.select", map[string]string{"workspace_id": target.Workspace}, nil); err != nil {
		return err
	}
	if err := rpc("surface.focus", map[string]string{"workspace_id": target.Workspace, "surface_id": target.Surface}, nil); err != nil {
		return err
	}
	return exec.CommandContext(ctx, "/usr/bin/open", "-a", "cmux").Run()
}
