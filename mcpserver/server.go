// Package mcpserver exposes the running application's inventory over loopback MCP.
package mcpserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"talkingduck/cli"
	"talkingduck/inventory"
)

const DefaultPort = 42389

type Backend interface {
	Paths() []string
	Snapshot() *inventory.Snapshot
	RefreshFolder(context.Context, string) error
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func compact(snapshot *inventory.Snapshot) (string, error) {
	text, err := cli.FormatAI(snapshot.Overview)
	if err != nil {
		return "", err
	}
	// A cached observation must not masquerade as a newly collected CLI result.
	for _, repo := range snapshot.Details.Repositories {
		if c := repo.Cache; c != nil {
			observed := "never"
			if !c.ObservedAt.IsZero() {
				observed = c.ObservedAt.Format(time.RFC3339)
			}
			text += fmt.Sprintf("\ncache %s %s observed=%s", repo.Path, c.State, observed)
			if c.Error != "" {
				text += " error=" + strings.Join(strings.Fields(c.Error), " ")
			}
		}
	}
	return text, nil
}

func resolveFolder(paths []string, input string) (string, error) {
	var matches []string
	for _, path := range paths {
		if input == path || (!filepath.IsAbs(input) && input == filepath.Base(path)) {
			matches = append(matches, path)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("folder must be a configured absolute path or unique folder name; got %q (%d matches)", input, len(matches))
	}
	return matches[0], nil
}

// Handler uses the official SDK's stateless Streamable HTTP transport.
func Handler(backend Backend) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "talkingduck", Version: "0.1.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "list_zoo_tasks", Description: "Return cached work inventory as compact CLI -ai text, including tickets, stages, primary session IDs and per-folder freshness. Does not scan. Unknown or stale state is not current activity."}, func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, any, error) {
		text, err := compact(backend.Snapshot())
		if err != nil {
			return nil, nil, err
		}
		return textResult(text), nil, nil
	})
	type refreshArgs struct {
		Folder string `json:"folder" jsonschema:"Configured absolute checkout path or unique folder name from list_zoo_tasks"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "refresh_folder", Description: "Refresh one configured checkout in the app's shared cache, wait for completion, update the window and tray, and return compact cached inventory. Reads local Git/spec/session data; does not modify the checkout."}, func(ctx context.Context, req *mcp.CallToolRequest, args refreshArgs) (*mcp.CallToolResult, any, error) {
		path, err := resolveFolder(backend.Paths(), args.Folder)
		if err != nil {
			return nil, nil, err
		}
		if err = backend.RefreshFolder(ctx, path); err != nil {
			return nil, nil, err
		}
		text, err := compact(backend.Snapshot())
		if err != nil {
			return nil, nil, err
		}
		return textResult(text), nil, nil
	})
	if folders, ok := backend.(FolderBackend); ok {
		addFolderTools(server, folders)
	}
	transport := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject browser origins and non-loopback Host values, including DNS rebinding.
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if (host != "127.0.0.1" && host != "localhost" && host != "[::1]" && host != "::1") || r.Header.Get("Origin") != "" {
			http.Error(w, "loopback clients without a browser Origin only", http.StatusForbidden)
			return
		}
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		transport.ServeHTTP(w, r)
	})
}

// Start cannot bind to any non-loopback interface. Port zero is useful in tests.
// Close stops the listener and active HTTP requests on application shutdown.
func Start(port int, backend Backend) (*http.Server, net.Addr, error) {
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, nil, err
	}
	server := &http.Server{Handler: Handler(backend), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	go server.Serve(listener)
	return server, listener.Addr(), nil
}
