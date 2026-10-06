package actions

import (
	"path/filepath"
	"talkingduck/launchers"
	"talkingduck/settings"
	"testing"
)

func TestTicketURLUsesSelectedSettingsAndReloads(t *testing.T) {
	store := &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	o := Opener{SettingsPath: store.Path}
	if _, err := o.TicketURL("DEV-1"); err == nil {
		t.Fatal("unconfigured account linked")
	}
	for _, slug := range []string{"first", "second"} {
		_, err := store.SetTracker(launchers.TrackerSelection{ID: "linear", URL: "https://linear.app/" + slug})
		if err != nil {
			t.Fatal(err)
		}
		url, err := o.TicketURL("DEV-1")
		if err != nil || url != "https://linear.app/"+slug+"/issue/DEV-1" {
			t.Fatal(url, err)
		}
	}
}
