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
	_, err := c.ExecuteCommand(command)
	return err
}

func (c *NvimClient) OpenQuickfixWindow() error {
	_, err := c.ExecuteCommand("copen")
	return err
}

func (c *NvimClient) ExecuteCommand(command string) (string, error) {
	// Input validation
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	// Normalize command (remove leading colon if present)
	normalizedCommand := command
	if strings.HasPrefix(command, ":") {
		normalizedCommand = command[1:]
	}

	// Clear Vim's error message variable first
	if _, err := c.remoteExpr("execute('let v:errmsg = \"\"')"); err != nil {
		return "", fmt.Errorf("failed to clear error message: %v", err)
	}

	// Execute command and capture output using execute() function
	output, err := c.remoteExpr(fmt.Sprintf("execute('%s')", c.escapeVimString(normalizedCommand)))
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v", err)
	}

	// Check for Vim errors by reading v:errmsg
	vimError, err := c.remoteExpr("v:errmsg")
	if err == nil && strings.TrimSpace(vimError) != "" {
		return "", fmt.Errorf("vim error: %s", vimError)
	}

	// Return the command output, or a success message if no output
	if strings.TrimSpace(output) == "" {
		return fmt.Sprintf("Command executed successfully: %s", command), nil
	}

	return output, nil
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

func (c *NvimClient) GetBufferContext() (string, error) {
	var result strings.Builder

	// Get file path
	filePath, err := c.remoteExpr("expand('%:p')")
	if err != nil {
		return "", fmt.Errorf("failed to get file path: %v", err)
	}
	result.WriteString("FILE_PATH:" + filePath + "\n")

	// Get cursor position
	cursor, err := c.remoteExpr("printf('%d:%d', line('.'), col('.'))")
	if err != nil {
		return "", fmt.Errorf("failed to get cursor position: %v", err)
	}
	result.WriteString("CURSOR:" + cursor + "\n")

	// Get current mode
	mode, err := c.remoteExpr("mode()")
	if err != nil {
		return "", fmt.Errorf("failed to get mode: %v", err)
	}
	result.WriteString("MODE:" + mode + "\n")

	// Check if in visual mode and get selection
	if strings.HasPrefix(mode, "v") || strings.HasPrefix(mode, "V") || mode == "\x16" { // \x16 is Ctrl-V
		// Get visual selection range using current selection positions
		visualRange, err := c.remoteExpr("printf('%d:%d to %d:%d', getpos('v')[1], getpos('v')[2], getpos('.')[1], getpos('.')[2])")
		if err != nil {
			return "", fmt.Errorf("failed to get visual range: %v", err)
		}
		result.WriteString("VISUAL_SELECTION:" + visualRange + "\n")

		// Get selected text using Lua for more reliable extraction
		selectedText, err := c.remoteExpr(`luaeval('(function()
			local start_pos = vim.fn.getpos("v")
			local end_pos = vim.fn.getpos(".")
			local start_line, start_col = start_pos[2], start_pos[3]
			local end_line, end_col = end_pos[2], end_pos[3]

			-- Ensure proper ordering
			if start_line > end_line or (start_line == end_line and start_col > end_col) then
				start_line, end_line = end_line, start_line
				start_col, end_col = end_col, start_col
			end

			local lines = vim.api.nvim_buf_get_lines(0, start_line - 1, end_line, false)
			return table.concat(lines, "\\n")
		end)()')`)
		if err != nil {
			return "", fmt.Errorf("failed to get selected text: %v", err)
		}
		result.WriteString("SELECTED_TEXT:" + selectedText + "\n")
	} else {
		// Get current line
		currentLine, err := c.remoteExpr("getline('.')")
		if err != nil {
			return "", fmt.Errorf("failed to get current line: %v", err)
		}
		result.WriteString("CURRENT_LINE:" + currentLine + "\n")
	}

	return result.String(), nil
}

func (c *NvimClient) GetDiagnostics() (string, error) {
	// Use Lua expression to get diagnostics as formatted string
	expr := `luaeval('(function()
		local diagnostics = vim.diagnostic.get(0)
		if #diagnostics == 0 then
			return "NO_DIAGNOSTICS"
		else
			local result = {}
			for _, diag in ipairs(diagnostics) do
				local severity_map = {"ERROR", "WARN", "INFO", "HINT"}
				local severity = severity_map[diag.severity] or "UNKNOWN"
				table.insert(result, "DIAGNOSTIC:" .. (diag.lnum + 1) .. ":" .. (diag.col + 1) .. ":" .. severity .. ":" .. (diag.message or ""))
			end
			return table.concat(result, "\\n")
		end
	end)()')`

	output, err := c.remoteExpr(expr)
	if err != nil {
		return "", fmt.Errorf("failed to get diagnostics: %v", err)
	}

	return output, nil
}

func (c *NvimClient) remoteExpr(expr string) (string, error) {
	cmd := exec.Command("nvim", "--server", c.socketPath, "--remote-expr", expr)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute expression: %v, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
