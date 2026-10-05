package mcpserver

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"strings"
	"talkingduck/settings"
)

type FolderBackend interface {
	ListFolders() ([]settings.FolderStatus, error)
	AddMonitoredFolder(string) error
	RemoveMonitoredFolder(string) error
	UpdateFolderMenubarItemShortcut(string, string) error
}

func folderText(backend FolderBackend) (*mcp.CallToolResult, any, error) {
	folders, err := backend.ListFolders()
	if err != nil {
		return nil, nil, err
	}
	lines := []string{}
	for _, f := range folders {
		visibility := "Hide"
		if f.Visible {
			visibility = "Show"
		}
		shortcut := f.MenubarItemShortcut
		if shortcut == "" {
			shortcut = "Hide"
		}
		source := "added"
		if f.Discovered {
			source = "discovered"
		}
		line := fmt.Sprintf("%s %s menubar=%s status=%s source=%s", f.Path, visibility, shortcut, f.Status, source)
		if f.Error != "" {
			line += " error=" + strings.Join(strings.Fields(f.Error), " ")
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "No folders configured.")
	}
	return textResult(strings.Join(lines, "\n")), nil, nil
}
func addFolderTools(server *mcp.Server, backend FolderBackend) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_folders", Description: "List all saved folders, including hidden/discovered ones, Show/Hide, menu bar shortcut and current cache or missing-folder status. Does not scan."}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
		return folderText(backend)
	})
	type folderArgs struct {
		Folder string `json:"folder" jsonschema:"Absolute folder path"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "add_monitored_folder", Description: "Save a folder as Show and start monitoring it. Existing hidden folders are re-enabled. Does not change repository files. Menu bar visibility is configured separately."}, func(_ context.Context, _ *mcp.CallToolRequest, args folderArgs) (*mcp.CallToolResult, any, error) {
		if err := backend.AddMonitoredFolder(args.Folder); err != nil {
			return nil, nil, err
		}
		return folderText(backend)
	})
	mcp.AddTool(server, &mcp.Tool{Name: "remove_monitored_folder", Description: "Set a folder to Hide and stop monitoring it. Retains its settings so discovery cannot re-enable it. Does not delete any files."}, func(_ context.Context, _ *mcp.CallToolRequest, args folderArgs) (*mcp.CallToolResult, any, error) {
		if err := backend.RemoveMonitoredFolder(args.Folder); err != nil {
			return nil, nil, err
		}
		return folderText(backend)
	})
	type shortcutArgs struct {
		Folder   string `json:"folder" jsonschema:"Saved absolute folder path"`
		Shortcut string `json:"shortcut" jsonschema:"Empty string hides the menu bar item; otherwise one uppercase/lowercase letter or number 0 through 19, stored as a string"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "update_folder_menubar_item_shortcut", Description: "Set the folder's menu bar label. Empty hides it; nonempty includes it when the folder is monitored. Case is preserved."}, func(_ context.Context, _ *mcp.CallToolRequest, args shortcutArgs) (*mcp.CallToolResult, any, error) {
		if err := backend.UpdateFolderMenubarItemShortcut(args.Folder, args.Shortcut); err != nil {
			return nil, nil, err
		}
		return folderText(backend)
	})
}
