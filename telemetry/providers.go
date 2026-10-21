package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type credentials struct{ token, account string }

func readCredentials(id string) (credentials, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return credentials{}, err
	}
	path := filepath.Join(home, "."+id, "auth.json")
	if id == "codex" {
		if h := os.Getenv("CODEX_HOME"); h != "" {
			path = filepath.Join(h, "auth.json")
		}
	}
	if id == "claude" {
		path = filepath.Join(home, ".claude", ".credentials.json")
		if h := os.Getenv("CLAUDE_CONFIG_DIR"); h != "" {
			path = filepath.Join(h, ".credentials.json")
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return credentials{}, err
	}
	defer f.Close()
	var data map[string]json.RawMessage
	if err = json.NewDecoder(io.LimitReader(f, 1<<20)).Decode(&data); err != nil {
		return credentials{}, err
	}
	var c credentials
	switch id {
	case "codex":
		var x struct {
			Access  string `json:"access_token"`
			Account string `json:"account_id"`
		}
		if err = json.Unmarshal(data["tokens"], &x); err != nil {
			return c, err
		}
		c.token, c.account = x.Access, x.Account
	case "claude":
		var x struct {
			Access string `json:"accessToken"`
		}
		if err = json.Unmarshal(data["claudeAiOauth"], &x); err != nil {
			return c, err
		}
		c.token = x.Access
	case "grok":
		// Grok stores one OAuth record per issuer/client. Do not pick randomly among accounts.
		if raw, ok := data["key"]; ok {
			json.Unmarshal(raw, &c.token)
		} else {
			for _, raw := range data {
				var x struct {
					Key string `json:"key"`
					ID  string `json:"user_id"`
				}
				if json.Unmarshal(raw, &x) == nil && x.Key != "" {
					if c.token != "" {
						return credentials{}, fmt.Errorf("multiple Grok credentials")
					}
					c.token, c.account = x.Key, x.ID
				}
			}
		}
	}
	if c.token == "" {
		return c, fmt.Errorf("no subscription credentials")
	}
	return c, nil
}
func collect(parent context.Context, id, mode string) Provider {
	p := Provider{ID: id, State: "unavailable", RecentlyUsed: recentlyUsed(id, time.Now())}
	if mode != "on" && !p.RecentlyUsed {
		return p
	}
	c, err := readCredentials(id)
	if err != nil {
		if !os.IsNotExist(err) {
			p.Error = "Local credentials unavailable"
		}
		return p
	}
	p.LoggedIn = true
	sum := sha256.Sum256([]byte(c.account + "\x00" + c.token))
	p.identity = hex.EncodeToString(sum[:])
	endpoints := map[string]string{"codex": "https://chatgpt.com/backend-api/wham/usage", "claude": "https://api.anthropic.com/api/oauth/usage", "grok": "https://cli-chat-proxy.grok.com/v1/billing?format=credits"}
	ctx, cancel := context.WithTimeout(parent, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoints[id], nil)
	if err != nil {
		p.State = "error"
		p.Error = "Cannot create usage request"
		return p
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if id == "codex" && c.account != "" {
		req.Header.Set("ChatGPT-Account-Id", c.account)
	}
	if id == "claude" {
		req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	}
	if id == "grok" {
		req.Header.Set("x-xai-token-auth", "xai-grok-cli")
	}
	// Never forward credentials to a redirect target.
	client := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		p.State = "error"
		p.Error = "Usage request failed"
		return p
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		p.retryAfter = retryDelay(resp.Header.Get("Retry-After"), time.Now())
		p.State = "error"
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			p.State = "auth"
		}
		p.Error = fmt.Sprintf("Usage request: HTTP %d", resp.StatusCode)
		return p
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		p.State = "error"
		p.Error = "Cannot read usage response"
		return p
	}
	remaining, reset, err := parseUsage(id, raw)
	if err != nil {
		p.State = "unavailable"
		p.Error = err.Error()
		return p
	}
	p.State = "ok"
	p.RemainingPercent = &remaining
	p.ResetsAt = reset
	p.ObservedAt = time.Now()
	return p
}
func remaining(used float64) (float64, error) {
	if math.IsNaN(used) || math.IsInf(used, 0) || used < 0 {
		return 0, fmt.Errorf("Invalid weekly usage")
	}
	return math.Max(0, 100-used), nil
}
func timestamp(raw json.RawMessage) *time.Time {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return &t
		}
	}
	var n float64
	if json.Unmarshal(raw, &n) == nil && n > 0 {
		t := time.Unix(int64(n), 0)
		return &t
	}
	return nil
}
func parseUsage(id string, raw []byte) (float64, *time.Time, error) {
	missing := fmt.Errorf("Weekly limit unavailable")
	switch id {
	case "codex":
		type window struct {
			Used    *float64        `json:"used_percent"`
			Seconds int             `json:"limit_window_seconds"`
			Reset   json.RawMessage `json:"reset_at"`
		}
		var x struct {
			Limit struct {
				P *window `json:"primary_window"`
				S *window `json:"secondary_window"`
			} `json:"rate_limit"`
		}
		if json.Unmarshal(raw, &x) != nil {
			return 0, nil, fmt.Errorf("Invalid usage response")
		}
		for _, w := range []*window{x.Limit.P, x.Limit.S} {
			if w != nil && w.Used != nil && w.Seconds == 604800 {
				r, e := remaining(*w.Used)
				return r, timestamp(w.Reset), e
			}
		}
	case "claude":
		var x struct {
			Week *struct {
				Used  *float64        `json:"utilization"`
				Reset json.RawMessage `json:"resets_at"`
			} `json:"seven_day"`
		}
		if json.Unmarshal(raw, &x) != nil {
			return 0, nil, fmt.Errorf("Invalid usage response")
		}
		if x.Week != nil && x.Week.Used != nil {
			r, e := remaining(*x.Week.Used)
			return r, timestamp(x.Week.Reset), e
		}
	case "grok":
		var x struct {
			Config struct {
				Used     *float64                             `json:"creditUsagePercent"`
				Cap      struct{ Val json.RawMessage }        `json:"onDemandCap"`
				Consumed struct{ Val json.RawMessage }        `json:"onDemandUsed"`
				Period   struct{ Start, End json.RawMessage } `json:"currentPeriod"`
				Start    json.RawMessage                      `json:"billingPeriodStart"`
				End      json.RawMessage                      `json:"billingPeriodEnd"`
			} `json:"config"`
		}
		if json.Unmarshal(raw, &x) != nil {
			return 0, nil, fmt.Errorf("Invalid usage response")
		}
		if x.Config.Used == nil {
			cap, capOK := number(x.Config.Cap.Val)
			used, usedOK := number(x.Config.Consumed.Val)
			if capOK && usedOK && cap > 0 && used >= 0 {
				percent := 100 * used / cap
				x.Config.Used = &percent
			}
		}
		start, end := timestamp(x.Config.Period.Start), timestamp(x.Config.Period.End)
		if start == nil || end == nil {
			start, end = timestamp(x.Config.Start), timestamp(x.Config.End)
		}
		if x.Config.Used != nil && start != nil && end != nil && end.Sub(*start) >= 167*time.Hour && end.Sub(*start) <= 169*time.Hour {
			r, e := remaining(*x.Config.Used)
			return r, end, e
		}
	}
	return 0, nil, missing
}

func number(raw json.RawMessage) (float64, bool) {
	var n float64
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	if json.Unmarshal(raw, &n) != nil {
		var s string
		if json.Unmarshal(raw, &s) != nil {
			return 0, false
		}
		var err error
		n, err = strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
	}
	return n, !math.IsNaN(n) && !math.IsInf(n, 0)
}

func retryDelay(value string, now time.Time) time.Duration {
	if n, err := strconv.ParseInt(value, 10, 64); err == nil && n > 0 {
		if n > 86400 {
			n = 86400
		}
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil && t.After(now) {
		d := t.Sub(now)
		if d > 24*time.Hour {
			d = 24 * time.Hour
		}
		return d
	}
	return 0
}
