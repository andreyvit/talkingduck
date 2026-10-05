package actions

import "testing"

func TestCurrentGrokHook(t *testing.T) {
	h := grokHook{SessionID: "one", PID: 42, Start: 100, Updated: 200, Surface: "surface"}
	for _, tc := range []struct {
		name    string
		records map[string]grokHook
		want    bool
	}{
		{"exact", map[string]grokHook{"one": h}, true},
		{"missing", nil, false},
		{"reused surface", map[string]grokHook{"one": h, "two": {PID: 43, Start: 110, Updated: 201, Surface: "surface"}}, false},
		{"changed session in process", map[string]grokHook{"one": h, "two": {PID: 42, Start: 100, Updated: 201, Surface: "other"}}, false},
		{"ambiguous time", map[string]grokHook{"one": h, "two": {Updated: 200, Surface: "surface"}}, false},
		{"older history", map[string]grokHook{"one": h, "two": {Updated: 199, Surface: "surface"}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := currentGrokHook(tc.records, "one")
			if (err == nil) != tc.want {
				t.Fatal(err)
			}
		})
	}
}
