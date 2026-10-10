package launchers

import (
	"context"
	"net/url"
	"os/exec"
	"reflect"
	"testing"
)

func TestNewAgentURLs(t *testing.T) {
	path := "/projects/a space/#one&two?'three"
	for _, tc := range []struct{ engine, host, key string }{{"codex", "threads", "path"}, {"claude", "code", "folder"}} {
		target, err := AgentURL(tc.engine, path)
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(target)
		if err != nil || u.Host != tc.host || u.Path != "/new" || u.Query().Get(tc.key) != path || len(u.Query()) != 1 {
			t.Fatal(target, err)
		}
	}
	if _, err := AgentURL("codex", "relative"); err == nil {
		t.Fatal("relative folder accepted")
	}
	if _, err := AgentURL("unknown", "/project"); err == nil {
		t.Fatal("unknown agent accepted")
	}
}

func TestGrokFreshSessionCommand(t *testing.T) {
	path := "/projects/a 'quoted' $(false); folder"
	executable := "/tools/Grok Build/grok"
	got := GrokCommand(path, executable, "cmux")
	want := []string{"new-workspace", "--name", "Grok · a 'quoted' $(false); folder", "--cwd", path, "--focus", "true", "--command",
		"'/tools/Grok Build/grok' --cwd '/projects/a '\\''quoted'\\'' $(false); folder'"}
	if got.Executable != "cmux" || !reflect.DeepEqual(got.Args, want) {
		t.Fatalf("%+v", got)
	}
	// Parse with a real shell while replacing execution with argument printing.
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no POSIX shell")
	}
	out, err := exec.CommandContext(context.Background(), shell, "-c", "printf '%s\\n' "+got.Args[len(got.Args)-1]).Output()
	if err != nil || string(out) != executable+"\n--cwd\n"+path+"\n" {
		t.Fatalf("%q %v", out, err)
	}
}
