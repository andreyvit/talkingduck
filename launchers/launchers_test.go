package launchers

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestCustomArgumentsRemainData(t *testing.T) {
	path := `/tmp/a b/$(touch nope);'x'`
	plan, err := Plan(Editors, Selection{ID: "custom", Path: "/bin/editor", Args: `--project "$PATH" --name 'two words' "" C:\Tools\editor`}, "linux", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"--project", path, "--name", "two words", "", `C:\Tools\editor`}
	if !reflect.DeepEqual(plan.Args, want) {
		t.Fatalf("%q", plan.Args)
	}
	if _, err = ParseArgs(`"unclosed`); err == nil {
		t.Fatal("accepted unmatched quote")
	}
	if _, err = Plan(Editors, Selection{ID: "custom", Path: "editor", Args: "$HOME"}, "linux", path, nil); err == nil {
		t.Fatal("accepted unsupported placeholder")
	}
}
func TestPlatformPlans(t *testing.T) {
	lookup := func(s string) (string, error) { return s, nil }
	for _, tc := range []struct {
		apps              []App
		id, platform, exe string
		args              []string
	}{
		{Editors, "zed", "darwin", "/usr/bin/open", []string{"-b", "dev.zed.Zed", "/project"}},
		{GitClients, "sublime-merge", "darwin", "/usr/bin/open", []string{"-b", "com.sublimemerge", "/project"}},
		{Editors, "vscode", "windows", "Code.exe", []string{"/project"}},
		{GitClients, "gitkraken", "linux", "gitkraken", []string{"-p", "/project"}},
	} {
		p, e := Plan(tc.apps, Selection{ID: tc.id}, tc.platform, "/project", lookup)
		if e != nil || p.Executable != tc.exe || !reflect.DeepEqual(p.Args, tc.args) {
			t.Fatalf("%s: %+v %v", tc.id, p, e)
		}
	}
	if _, e := Plan(GitClients, Selection{ID: "fork"}, "linux", "/project", lookup); e == nil {
		t.Fatal("missing platform accepted")
	}
	if _, e := Plan(Editors, Selection{ID: "zed"}, "linux", "/project", func(string) (string, error) { return "", fmt.Errorf("missing") }); e == nil {
		t.Fatal("missing launcher accepted")
	}
	for _, a := range Available(GitClients, "linux") {
		if a.ID == "fork" {
			t.Fatal("mac-only app offered")
		}
	}
}
func TestTrackerPastedURLs(t *testing.T) {
	for _, tc := range []struct{ id, input, ticket, want, account string }{
		{"linear", "https://linear.app/acme/issue/DEV-1/title", "DEV-2", "https://linear.app/acme/issue/DEV-2", "acme"},
		{"jira", "https://acme.atlassian.net/browse/DEV-1", "DEV-2", "https://acme.atlassian.net/browse/DEV-2", "acme"},
		{"github", "https://github.com/acme/app/issues/1", "2", "https://github.com/acme/app/issues/2", "acme/app"},
		{"gitlab", "https://gitlab.com/acme/sub/app/-/issues/1", "2", "https://gitlab.com/acme/sub/app/-/issues/2", "acme/sub/app"},
		{"azure", "https://dev.azure.com/acme/app/_workitems/edit/1", "2", "https://dev.azure.com/acme/app/_workitems/edit/2", "acme/app"},
		{"youtrack", "https://acme.youtrack.cloud/issue/DEV-1", "DEV-2", "https://acme.youtrack.cloud/issue/DEV-2", "https://acme.youtrack.cloud"},
		{"redmine", "https://issues.example.com/redmine/issues/1", "2", "https://issues.example.com/redmine/issues/2", "https://issues.example.com/redmine"},
		{"shortcut", "https://app.shortcut.com/acme/story/1", "2", "https://app.shortcut.com/acme/story/2", "acme"},
		{"trello", "", "abc123", "https://trello.com/c/abc123", ""},
	} {
		s := TrackerSelection{ID: tc.id, URL: tc.input, AccountSlug: "untrusted"}
		c, e := s.Configure()
		if e != nil || c.AccountSlug != tc.account {
			t.Fatalf("%s: %+v %v", tc.id, c, e)
		}
		got, e := c.TicketLink(tc.ticket)
		if e != nil || got != tc.want {
			t.Fatalf("%s: %s %v", tc.id, got, e)
		}
	}
	for _, input := range []string{"https://linear.app.evil/acme", "https://example.com/acme", "https://linear.app/", "javascript:alert(1)"} {
		if _, e := (TrackerSelection{ID: "linear", URL: input}).Configure(); e == nil {
			t.Fatalf("accepted %s", input)
		}
	}
}
func TestCustomTracker(t *testing.T) {
	s := TrackerSelection{ID: "custom", Custom: Tracker{Name: "Private", TicketURL: "https://issues.example.com/find?id=$TICKET"}}
	got, e := s.TicketLink("A&B/#1")
	if e != nil || !strings.Contains(got, "A%26B%2F%231") {
		t.Fatalf("%s %v", got, e)
	}
	s.Custom.URLRegExp = "["
	if _, e = s.Configure(); e == nil {
		t.Fatal("bad regex accepted")
	}
	s.Custom.URLRegExp = ".*"
	if _, e = s.Configure(); e == nil {
		t.Fatal("missing group accepted")
	}
}
