package launchers

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type Tracker struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TicketURL string `json:"ticketURL"`
	RootURL   string `json:"rootURL"`
	URLRegExp string `json:"urlRegExp"`
}
type TrackerSelection struct {
	ID          string  `json:"id"`
	URL         string  `json:"url,omitempty"`
	AccountSlug string  `json:"account_slug,omitempty"`
	Custom      Tracker `json:"custom,omitempty"`
}

var Trackers = []Tracker{
	{"linear", "Linear", "https://linear.app/$ACCOUNT_SLUG/issue/$TICKET", "https://linear.app/$ACCOUNT_SLUG", `^https://linear\.app/([A-Za-z0-9_-]+)(?:/.*)?$`},
	{"jira", "Jira Cloud", "https://$ACCOUNT_SLUG.atlassian.net/browse/$TICKET", "https://$ACCOUNT_SLUG.atlassian.net", `^https://([A-Za-z0-9-]+)\.atlassian\.net(?:/.*)?$`},
	{"jira-server", "Jira Server / Data Center", "$ACCOUNT_SLUG/browse/$TICKET", "$ACCOUNT_SLUG", `^(https?://[^/?#]+(?:/jira)?)(?:/.*)?$`},
	{"github", "GitHub Issues", "https://github.com/$ACCOUNT_SLUG/issues/$TICKET", "https://github.com/$ACCOUNT_SLUG", `^https://github\.com/([^/?#]+/[^/?#]+)(?:/.*)?$`},
	{"gitlab", "GitLab Issues", "https://gitlab.com/$ACCOUNT_SLUG/-/issues/$TICKET", "https://gitlab.com/$ACCOUNT_SLUG", `^https://gitlab\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*?)(?:/-/.*)?$`},
	{"azure", "Azure DevOps", "https://dev.azure.com/$ACCOUNT_SLUG/_workitems/edit/$TICKET", "https://dev.azure.com/$ACCOUNT_SLUG", `^https://dev\.azure\.com/([^/?#]+/[^/?#]+)(?:/.*)?$`},
	{"youtrack", "YouTrack", "$ACCOUNT_SLUG/issue/$TICKET", "$ACCOUNT_SLUG", `^(https?://[^/?#]+(?:/youtrack)?)(?:/.*)?$`},
	{"redmine", "Redmine", "$ACCOUNT_SLUG/issues/$TICKET", "$ACCOUNT_SLUG", `^(https?://[^/?#]+(?:/redmine)?)(?:/.*)?$`},
	{"shortcut", "Shortcut", "https://app.shortcut.com/$ACCOUNT_SLUG/story/$TICKET", "https://app.shortcut.com/$ACCOUNT_SLUG", `^https://app\.shortcut\.com/([^/?#]+)(?:/.*)?$`},
	{"trello", "Trello", "https://trello.com/c/$TICKET", "https://trello.com", ""},
	{ID: "custom", Name: "Custom"},
}

func (s TrackerSelection) Profile() (Tracker, error) {
	if s.ID == "custom" {
		return s.Custom, nil
	}
	for _, p := range Trackers {
		if p.ID == s.ID {
			return p, nil
		}
	}
	return Tracker{}, fmt.Errorf("unknown ticket tracker %q", s.ID)
}
func webURL(s string) error {
	u, e := url.Parse(s)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return fmt.Errorf("expected an http or https URL")
	}
	return nil
}

// Configure validates a pasted URL and derives account_slug from capture group 1.
// It never trusts an account_slug supplied by a client.
func (s TrackerSelection) Configure() (TrackerSelection, error) {
	p, err := s.Profile()
	if err != nil {
		return s, err
	}
	s.AccountSlug = ""
	s.URL = strings.TrimRight(strings.TrimSpace(s.URL), "/")
	if !strings.Contains(p.TicketURL, "$TICKET") {
		return s, fmt.Errorf("ticket URL must contain $TICKET")
	}
	for _, template := range []string{p.TicketURL, p.RootURL} {
		if _, err := expand(template, map[string]string{"TICKET": "EX-1", "ACCOUNT_SLUG": "example"}); err != nil {
			return s, err
		}
	}
	needsAccount := strings.Contains(p.TicketURL+p.RootURL, "$ACCOUNT_SLUG")
	if needsAccount && p.URLRegExp == "" {
		return s, fmt.Errorf("an account URL regexp with capture group 1 is required")
	}
	if p.URLRegExp != "" {
		re, err := regexp.Compile(p.URLRegExp)
		if err != nil {
			return s, fmt.Errorf("invalid URL regexp: %w", err)
		}
		if re.NumSubexp() < 1 {
			return s, fmt.Errorf("URL regexp needs capture group 1 for $ACCOUNT_SLUG")
		}
		if s.URL == "" {
			return s, fmt.Errorf("paste a %s workspace or project URL", p.Name)
		}
		if err := webURL(s.URL); err != nil {
			return s, err
		}
		m := re.FindStringSubmatch(s.URL)
		if len(m) < 2 || m[0] != s.URL || m[1] == "" {
			return s, fmt.Errorf("URL does not match %s; paste its workspace or project URL", p.Name)
		}
		s.AccountSlug = m[1]
	}
	for _, template := range []string{p.TicketURL, p.RootURL} {
		if template == "" {
			continue
		}
		out, _ := expand(template, map[string]string{"TICKET": "EX-1", "ACCOUNT_SLUG": s.AccountSlug})
		if err := webURL(out); err != nil {
			return s, err
		}
	}
	return s, nil
}

// TicketLink escapes the ticket as one URL component; tracker IDs may be numeric or named.
func (s TrackerSelection) TicketLink(ticket string) (string, error) {
	if ticket == "" || ticket == "--" || strings.ContainsAny(ticket, "\r\n\x00") {
		return "", fmt.Errorf("invalid ticket reference")
	}
	configured, err := s.Configure()
	if err != nil {
		return "", err
	}
	p, _ := s.Profile()
	return expand(p.TicketURL, map[string]string{"TICKET": strings.ReplaceAll(url.QueryEscape(ticket), "+", "%20"), "ACCOUNT_SLUG": configured.AccountSlug})
}
