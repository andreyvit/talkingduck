package inventory

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildReturnsCorrelatedPartialSnapshot(t *testing.T) {
	root := repo(t)
	write(t, root, ".spec/current.md", "---\nstatus: planning\n---\n# Example\n\n## User request\nDEV-1234\n")
	paths := []string{root, filepath.Join(t.TempDir(), "missing")}
	before := append([]string(nil), paths...)
	snapshot, err := Build(Options{Paths: paths, NoAgents: true})
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Failed() || len(snapshot.Details.Repositories) != 2 || snapshot.Details.Repositories[0].Error != "" {
		t.Fatal(snapshot)
	}
	if !reflect.DeepEqual(paths, before) {
		t.Fatal("caller paths mutated")
	}
	repos := objects(snapshot.Overview["repositories"])
	specs := objects(repos[0]["specs"])
	if len(specs) != 1 || obj(specs[0]["workflow"])["stage"] != "HL" || specs[0]["primary_session"] != nil {
		t.Fatal(specs)
	}
	if obj(repos[1]["git"])["error"] == "" {
		t.Fatal("missing checkout error omitted")
	}
	// A new build must not share the previously returned mutable report.
	specs[0]["display_title"] = "changed by caller"
	next, err := Build(Options{Paths: []string{root}, NoAgents: true})
	if err != nil || next.Failed() {
		t.Fatal(next, err)
	}
	if objects(objects(next.Overview["repositories"])[0]["specs"])[0]["display_title"] == "changed by caller" {
		t.Fatal("snapshots share data")
	}
}

func TestBuildDefaultPathsAndValidation(t *testing.T) {
	root := t.TempDir()
	snapshot, err := Build(Options{Root: root, NoAgents: true})
	if err != nil || len(snapshot.Details.Repositories) != 10 {
		t.Fatal(snapshot, err)
	}
	if snapshot.Details.Repositories[0].Path != filepath.Join(root, "backend") || snapshot.Details.Repositories[9].Path != filepath.Join(root, "back10") {
		t.Fatal(snapshot)
	}
	if _, err = Build(Options{AgentLimit: -1}); err == nil {
		t.Fatal("negative limit accepted")
	}
	if _, err = CollectSessions(root, nil, -1); err == nil {
		t.Fatal("negative raw session limit accepted")
	}
}
