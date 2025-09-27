# Neovim MCP Server

A Model Context Protocol (MCP) server that connects AI agents to your live Neovim editing session. Agents can see your current context (what you're working on, current errors) and send analysis results back to your editor's quickfix list.

## Features

This MCP server provides 4 focused tools that enable smooth context sharing between you and AI agents:

1. **get_buffer_context** - Lets agents see what file you're in, your cursor position, and any selected text
2. **get_diagnostics** - Provides agents with current LSP errors, warnings, and hints from your code
3. **populate_quickfix** - Allows agents to send analysis results back to your editor as a navigable quickfix list
4. **execute_command** - Enables agents to run specific Vim commands when needed (returns command output)

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

**You**: "What does this function do?"

The agent will use `get_buffer_context` to see exactly what code you're looking at (your cursor position or selected text), then explain the functionality based on your current context in Neovim.

**You**: "I need to trace how the `validateLogin` function works. Can you help me navigate through the code and show me all the related functions?"

The agent will search for related functions like `hashPassword`, `checkUserExists`, then use `populate_quickfix` to create a navigation list of all related functions, allowing you to jump between them with `:cnext` and `:cprev`.


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
