package mcpserver

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"talkingduck/cli"
	"talkingduck/inventory"
)

type fakeBackend struct {
	mu        sync.Mutex
	refreshed []string
}

func (*fakeBackend) Paths() []string { return []string{"/example/one", "/example/two"} }
func (*fakeBackend) Snapshot() *inventory.Snapshot {
	return &inventory.Snapshot{Overview: inventory.Object{"repositories": []any{map[string]any{"path": "/example/one", "git": map[string]any{"cleanliness": "CLEAN"}}}}, Details: inventory.Report{Repositories: []inventory.Repo{{Path: "/example/one", Cache: &inventory.RefreshState{State: "pending"}}}}}
}
func (f *fakeBackend) RefreshFolder(ctx context.Context, path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshed = append(f.refreshed, path)
	return nil
}

func TestHTTPTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	backend := &fakeBackend{}
	server, addr, err := Start(0, backend)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if !addr.(*net.TCPAddr).IP.IsLoopback() {
		t.Fatalf("not loopback: %s", addr)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://" + addr.String() + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	listed, err := session.ListTools(ctx, nil)
	if err != nil || len(listed.Tools) != 2 {
		t.Fatalf("list: %v %v", listed, err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_inventory", Arguments: map[string]any{}})
	if err != nil || result.IsError {
		t.Fatalf("get: %v %v", result, err)
	}
	want, _ := cli.FormatAI(backend.Snapshot().Overview)
	if len(result.Content) != 1 || result.Content[0].(*mcp.TextContent).Text != want+"\ncache /example/one pending observed=never" {
		t.Fatalf("unexpected text: %v", result)
	}
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "refresh_folder", Arguments: map[string]any{"folder": "two"}})
	if err != nil || result.IsError {
		t.Fatalf("refresh: %v %v", result, err)
	}
	backend.mu.Lock()
	if len(backend.refreshed) != 1 || backend.refreshed[0] != "/example/two" {
		t.Fatal(backend.refreshed)
	}
	backend.mu.Unlock()
	result, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "refresh_folder", Arguments: map[string]any{"folder": "/unconfigured"}})
	if err != nil || !result.IsError {
		t.Fatalf("expected tool error: %v %v", result, err)
	}
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if len(backend.refreshed) != 1 {
		t.Fatal("unconfigured folder scanned")
	}
}

func TestHTTPRejectsExternalHostAndBrowserOrigin(t *testing.T) {
	for _, tc := range []struct{ host, origin string }{{"evil.example", ""}, {"127.0.0.1:42389", "https://evil.example"}, {"localhost:42389", "null"}} {
		r := httptest.NewRequest("POST", "http://localhost/mcp", strings.NewReader("{}"))
		r.Host = tc.host
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		w := httptest.NewRecorder()
		Handler(&fakeBackend{}).ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("%+v: %d", tc, w.Code)
		}
	}
}

func TestFolderResolutionRejectsAmbiguity(t *testing.T) {
	if _, err := resolveFolder([]string{"/a/repo", "/b/repo"}, "repo"); err == nil {
		t.Fatal("ambiguous basename accepted")
	}
	if _, err := resolveFolder([]string{"/a/repo"}, "../repo"); err == nil {
		t.Fatal("arbitrary relative path accepted")
	}
}
