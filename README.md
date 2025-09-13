# Neovim MCP Server

A Model Context Protocol (MCP) server that enables Claude to interact with your Neovim editor.

## Features

This MCP server provides two main tools:

1. **populate_quickfix** - Populate Neovim's quickfix list with code analysis results, errors, or warnings
2. **execute_command** - Execute any Vim command in the connected Neovim instance

## Installation

1. Clone or download this repository
2. Build the server:
   ```bash
   go build -o build/neovim-mcp
   ```


## Setup

### 1. Start Neovim with Socket Support

```bash
# Navigate to your project directory and start Neovim with a socket
cd /path/to/your/project
nvim --listen ~/.cache/nvim/$(basename "$PWD").sock
```

See [Neovim Remote Documentation](https://neovim.io/doc/user/remote.html) for more details.

### 2. Configure Claude Code

Add this server to your Claude Code configuration:

```bash
claude mcp add neovim /path/to/neovim-mcp/build/neovim-mcp --scope user
```

## Usage Examples

### Tracing a Feature Implementation

**Scenario**: You're working on a user authentication system and want to understand how login validation works.

**You**: "I need to trace how the `validateLogin` function works. Can you help me navigate through the code and show me all the related functions?"

**Claude will**:
1. Search for `validateLogin` function definition
2. Use `execute_command` to jump to the function: `:goto 156`
3. Find related functions like `hashPassword`, `checkUserExists`
4. Use `populate_quickfix` to create a navigation list of all related functions:
   ```json
   {
     "items": [
       {
         "filename": "/src/auth/login.go",
         "line": 156,
         "text": "validateLogin function definition",
         "type": "I"
       },
       {
         "filename": "/src/auth/crypto.go",
         "line": 23,
         "text": "hashPassword helper function",
         "type": "I"
       },
       {
         "filename": "/src/db/users.go",
         "line": 78,
         "text": "checkUserExists database call",
         "type": "I"
       }
     ]
   }
   ```

### Code Review and Issue Detection

**You**: "Can you review this pull request and highlight any potential issues in the quickfix list?"

**Claude will**:
1. Analyze the changed files for common issues
2. Populate the quickfix list with findings like unused variables, potential null pointer dereferences, or style violations
3. Use `execute_command` to navigate between issues: `cnext`, `cprev`


## Socket Detection

The server automatically detects Neovim sockets using:
1. `$NVIM` environment variable (when inside Neovim)
2. Working directory name: `~/.cache/nvim/{directory-name}.sock`


## Troubleshooting

1. **"No Neovim instance found"**: Make sure Neovim is running with a socket in the expected location
2. **Commands not executing**: Verify that the socket path is correct and Neovim is responsive
3. **Permission errors**: Ensure the socket file is accessible

## Requirements

- Go 1.19+ and Neovim with RPC support
