package main

import (
	"log"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Initialize the Neovim toolbox
	nvimToolbox, err := NewNvimToolbox()
	if err != nil {
		log.Printf("Warning during initialization: %v", err)
	}

	// Create MCP server with tool capabilities
	s := server.NewMCPServer(
		"neovim-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithInstructions("This MCP server provides access to the user's live Neovim editing session. Use get_buffer_context first to see what code the user is currently working on, get_diagnostics to understand any issues, and populate_quickfix to send your analysis results back to their editor."),
	)

	// Register tools
	nvimToolbox.RegisterTools(s)

	// Start the server
	log.Println("Starting Neovim MCP server...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}


