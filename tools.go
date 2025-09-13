package main

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NvimToolbox holds the client connection and implements tool handlers
type NvimToolbox struct {
	client *NvimClient
}

// NewNvimToolbox creates a new toolbox instance with Neovim client
func NewNvimToolbox() (*NvimToolbox, error) {
	client, err := NewNvimClient()
	if err != nil {
		log.Printf("Warning: %v", err)
		// Continue anyway - the client might connect later
		client = &NvimClient{}
	}

	return &NvimToolbox{
		client: client,
	}, nil
}

// RegisterTools creates and registers all MCP tools with the server
func (t *NvimToolbox) RegisterTools(s *server.MCPServer) {
	// Create populate_quickfix tool
	populateQuickfixTool := mcp.NewTool(
		"populate_quickfix",
		mcp.WithDescription("Populate Neovim's quickfix list with code analysis results or errors"),
		mcp.WithInputSchema[PopulateQuickfixArgs](),
	)

	// Create execute_command tool
	executeCommandTool := mcp.NewTool(
		"execute_command",
		mcp.WithDescription("Execute a Vim command in the connected Neovim instance"),
		mcp.WithInputSchema[ExecuteCommandArgs](),
	)

	// Register tools with their handlers
	s.AddTool(populateQuickfixTool, t.PopulateQuickfix)
	s.AddTool(executeCommandTool, t.ExecuteCommand)
}

// PopulateQuickfix populates Neovim's quickfix list with code analysis results or errors
func (t *NvimToolbox) PopulateQuickfix(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.ensureConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var args PopulateQuickfixArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to bind arguments: %v", err)), nil
	}

	// Build quickfix list from typed arguments
	var qfList []QuickfixItem
	for _, item := range args.Items {
		qfEntry := QuickfixItem{
			Filename: item.Filename,
			Line:     item.Line,
			Column:   item.Column,
			Text:     item.Text,
			Type:     item.Type,
		}
		qfList = append(qfList, qfEntry)
	}

	// Set quickfix list
	if err := t.client.SetQuickfixList(qfList); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to set quickfix list: %v", err)), nil
	}

	// Open quickfix window
	if err := t.client.OpenQuickfixWindow(); err != nil {
		log.Printf("Warning: Could not open quickfix window: %v", err)
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully populated quickfix list with %d items", len(qfList))), nil
}

// ExecuteCommand executes a Vim command in the connected Neovim instance
func (t *NvimToolbox) ExecuteCommand(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.ensureConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var args ExecuteCommandArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to bind arguments: %v", err)), nil
	}

	if err := t.client.ExecuteCommand(args.Command); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute command: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully executed command: %s", args.Command)), nil
}

// ensureConnection tries to reconnect to Neovim if not already connected
func (t *NvimToolbox) ensureConnection() error {
	if t.client.socketPath == "" {
		client, err := NewNvimClient()
		if err != nil {
			return fmt.Errorf("no Neovim instance found: %w", err)
		}
		t.client = client
	}
	return nil
}

// Tool argument structs for typed schemas
type QuickfixItemArg struct {
	Filename string `json:"filename" jsonschema:"description=File path"`
	Line     int    `json:"line" jsonschema:"description=Line number"`
	Column   int    `json:"column,omitempty" jsonschema:"description=Column number (optional)"`
	Text     string `json:"text" jsonschema:"description=Error or warning message"`
	Type     string `json:"type,omitempty" jsonschema:"description=Type of entry (E for error W for warning I for info),enum=E,enum=W,enum=I"`
}

type PopulateQuickfixArgs struct {
	Items []QuickfixItemArg `json:"items" jsonschema:"description=Array of quickfix items"`
}

type ExecuteCommandArgs struct {
	Command string `json:"command" jsonschema:"description=Vim command to execute (e.g. 'set number' 'vsplit' 'wq' etc.)"`
}
