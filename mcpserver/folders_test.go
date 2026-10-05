package mcpserver

import (
	"context"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"path/filepath"
	"strings"
	"talkingduck/settings"
	"testing"
	"time"
)

type foldersBackend struct {
	fakeBackend
	store *settings.Store
}

func (b *foldersBackend) ListFolders() ([]settings.FolderStatus, error) {
	d, e := b.store.Load()
	r := []settings.FolderStatus{}
	for _, f := range d.Folders {
		state := "hidden"
		if f.Visible {
			state = "pending"
		}
		r = append(r, settings.FolderStatus{Folder: f, Status: state})
	}
	return r, e
}
func (b *foldersBackend) AddMonitoredFolder(p string) error {
	_, e := b.store.SetVisible(p, true)
	return e
}
func (b *foldersBackend) RemoveMonitoredFolder(p string) error {
	_, e := b.store.SetVisible(p, false)
	return e
}
func (b *foldersBackend) UpdateFolderMenubarItemShortcut(p, s string) error {
	_, e := b.store.SetShortcut(p, s)
	return e
}
func TestFolderTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	backend := &foldersBackend{store: &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}}
	server, addr, err := Start(0, backend)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://" + addr.String() + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 6 {
		t.Fatal(tools, err)
	}
	folder := t.TempDir()
	call := func(name string, args map[string]any, want string) {
		t.Helper()
		result, e := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if e != nil || result.IsError {
			t.Fatal(result, e)
		}
		if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, want) {
			t.Fatal(result)
		}
	}
	call("list_folders", map[string]any{}, "No folders")
	call("add_monitored_folder", map[string]any{"folder": folder}, "Show menubar=Hide status=pending")
	call("update_folder_menubar_item_shortcut", map[string]any{"folder": folder, "shortcut": "19"}, "menubar=19")
	call("remove_monitored_folder", map[string]any{"folder": folder}, "Hide menubar=19 status=hidden")
	call("list_folders", map[string]any{}, folder)
	result, e := session.CallTool(ctx, &mcp.CallToolParams{Name: "update_folder_menubar_item_shortcut", Arguments: map[string]any{"folder": folder, "shortcut": "20"}})
	if e != nil || !result.IsError {
		t.Fatal(result, e)
	}
}
