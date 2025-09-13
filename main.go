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
	)

	// Register tools
	nvimToolbox.RegisterTools(s)

	// Start the server
	log.Println("Starting Neovim MCP server...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatal(err)
	}
}


