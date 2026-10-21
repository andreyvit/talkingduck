package settings

import (
	"fmt"
	"net"
	"strings"
)

// Empty values preserve defaults for settings files written before monitoring.
type Menubar struct {
	Ping       string `json:"ping,omitempty"`
	PingTarget string `json:"ping_target,omitempty"`
	Codex      string `json:"codex,omitempty"`
	Claude     string `json:"claude,omitempty"`
	Grok       string `json:"grok,omitempty"`
	Projects   string `json:"projects,omitempty"`
}

func (m Menubar) Defaults() Menubar {
	if m.Ping == "" {
		m.Ping = "vertical"
	}
	if m.PingTarget == "" {
		m.PingTarget = "8.8.8.8"
	}
	if m.Codex == "" {
		m.Codex = "auto"
	}
	if m.Claude == "" {
		m.Claude = "auto"
	}
	if m.Grok == "" {
		m.Grok = "auto"
	}
	if m.Projects == "" {
		m.Projects = "on"
	}
	return m
}
func (m Menubar) Validate() error {
	m = m.Defaults()
	if m.Ping != "vertical" && m.Ping != "off" {
		return fmt.Errorf("invalid ping presentation")
	}
	if m.Projects != "on" && m.Projects != "off" {
		return fmt.Errorf("invalid projects presentation")
	}
	for _, mode := range []string{m.Codex, m.Claude, m.Grok} {
		if mode != "on" && mode != "auto" && mode != "off" {
			return fmt.Errorf("limits must be On, Auto, or Off")
		}
	}
	target := m.PingTarget
	if net.ParseIP(target) != nil {
		return nil
	}
	if len(target) > 253 || strings.TrimSpace(target) != target {
		return fmt.Errorf("ping server must be an IP address or hostname")
	}
	for _, label := range strings.Split(strings.TrimSuffix(target, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("invalid ping hostname")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return fmt.Errorf("ping server must be an IP address or hostname")
			}
		}
	}
	return nil
}
func (s *Store) SetMenubar(m Menubar) (Data, error) {
	m = m.Defaults()
	if err := m.Validate(); err != nil {
		return Data{}, err
	}
	return s.update(func(d *Data) error { d.Menubar = m; return nil })
}
