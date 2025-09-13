package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type NvimClient struct {
	socketPath string
}

func NewNvimClient() (*NvimClient, error) {
	// Use auto-detection
	socketPath := findNvimSocket()
	if socketPath == "" {
		return nil, fmt.Errorf("no Neovim instance found for current directory")
	}

	return &NvimClient{
		socketPath: socketPath,
	}, nil
}

func findNvimSocket() string {
	// Check if NVIM environment variable is set (when running inside nvim)
	if nvimSocket := os.Getenv("NVIM"); nvimSocket != "" {
		if _, err := os.Stat(nvimSocket); err == nil {
			return nvimSocket
		}
	}

	// Otherwise, try to find the socket for the current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Generate socket path using working directory name
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".cache")
	}
	sockDir := filepath.Join(cacheDir, "nvim")

	projBase := filepath.Base(pwd)
	socketPath := filepath.Join(sockDir, fmt.Sprintf("%s.sock", projBase))

	// Check if socket exists
	if _, err := os.Stat(socketPath); err == nil {
		return socketPath
	}

	return ""
}

func (c *NvimClient) SetQuickfixList(items []QuickfixItem) error {
	// Convert items to Vim dictionary format
	vimList := c.quickfixItemsToVimList(items)

	// Use setqflist() function
	command := fmt.Sprintf("call setqflist(%s)", vimList)
	return c.ExecuteCommand(command)
}

func (c *NvimClient) OpenQuickfixWindow() error {
	return c.ExecuteCommand("copen")
}

func (c *NvimClient) ExecuteCommand(command string) error {
	// Ensure we're in normal mode before executing the command
	// Send multiple escapes to handle different scenarios:
	// - <C-\><C-n> forces normal mode from any mode (including terminal)
	// - <Esc><Esc> handles insert mode and visual mode
	// - The final : enters command mode
	escapeSequence := `<C-\><C-n><Esc><Esc>:`
	remoteCmd := fmt.Sprintf("%s%s<CR>", escapeSequence, command)

	cmd := exec.Command("nvim", "--server", c.socketPath, "--remote-send", remoteCmd)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute command: %v, stderr: %s", err, stderr.String())
	}

	return nil
}

func (c *NvimClient) quickfixItemsToVimList(items []QuickfixItem) string {
	var itemStrs []string

	for _, item := range items {
		parts := []string{}

		// Add filename
		parts = append(parts, fmt.Sprintf("'filename': '%s'", c.escapeVimString(item.Filename)))

		// Add line number
		parts = append(parts, fmt.Sprintf("'lnum': %d", item.Line))

		// Add column if specified
		if item.Column > 0 {
			parts = append(parts, fmt.Sprintf("'col': %d", item.Column))
		}

		// Add text
		parts = append(parts, fmt.Sprintf("'text': '%s'", c.escapeVimString(item.Text)))

		// Add type if specified
		if item.Type != "" {
			parts = append(parts, fmt.Sprintf("'type': '%s'", item.Type))
		}

		itemStrs = append(itemStrs, "{"+strings.Join(parts, ", ")+"}")
	}

	return "[" + strings.Join(itemStrs, ", ") + "]"
}

func (c *NvimClient) escapeVimString(s string) string {
	// Escape single quotes for Vim strings
	return strings.ReplaceAll(s, "'", "''")
}

type QuickfixItem struct {
	Filename string
	Line     int
	Column   int
	Text     string
	Type     string // "E" for error, "W" for warning, "I" for info
}
