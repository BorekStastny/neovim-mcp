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
		mcp.WithDescription("Send analysis results, errors, or findings back to the user's editor as a navigable list. Use this to show the user locations that need attention."),
		mcp.WithInputSchema[PopulateQuickfixArgs](),
	)

	// Create execute_command tool
	executeCommandTool := mcp.NewTool(
		"execute_command",
		mcp.WithDescription("Execute Vim commands when you need specific editor information not available through other tools. Prefer the dedicated context tools first."),
		mcp.WithInputSchema[ExecuteCommandArgs](),
	)

	// Create get_buffer_context tool
	getBufferContextTool := mcp.NewTool(
		"get_buffer_context",
		mcp.WithDescription("Get what the user is currently looking at - file path, cursor position, selected text, and current line. Use this first to understand what code the user wants help with."),
		mcp.WithInputSchema[GetBufferContextArgs](),
	)

	// Create get_diagnostics tool
	getDiagnosticsTool := mcp.NewTool(
		"get_diagnostics",
		mcp.WithDescription("Get current errors, warnings, and hints from language servers. Use this to understand what's broken or needs attention in the code."),
		mcp.WithInputSchema[GetDiagnosticsArgs](),
	)

	// Register tools with their handlers
	s.AddTool(populateQuickfixTool, t.PopulateQuickfix)
	s.AddTool(executeCommandTool, t.ExecuteCommand)
	s.AddTool(getBufferContextTool, t.GetBufferContext)
	s.AddTool(getDiagnosticsTool, t.GetDiagnostics)
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

	output, err := t.client.ExecuteCommand(args.Command)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute command: %v", err)), nil
	}

	return mcp.NewToolResultText(output), nil
}

// GetBufferContext retrieves current buffer context including cursor position and visual selection
func (t *NvimToolbox) GetBufferContext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.ensureConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var args GetBufferContextArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to bind arguments: %v", err)), nil
	}

	context, err := t.client.GetBufferContext()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get buffer context: %v", err)), nil
	}

	return mcp.NewToolResultText(context), nil
}

// GetDiagnostics retrieves LSP diagnostics for the current buffer
func (t *NvimToolbox) GetDiagnostics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := t.ensureConnection(); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var args GetDiagnosticsArgs
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to bind arguments: %v", err)), nil
	}

	diagnostics, err := t.client.GetDiagnostics()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get diagnostics: %v", err)), nil
	}

	return mcp.NewToolResultText(diagnostics), nil
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

type GetBufferContextArgs struct {
	// No arguments needed - will return current line or visual selection
}

type GetDiagnosticsArgs struct {
	// No arguments needed for now
}
