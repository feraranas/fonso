# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Fonso is a Go-based AI coding agent framework that demonstrates building agents with Claude using the Anthropic API. It provides multiple command-line tools showcasing different agent architectures, from basic chat to full-featured coding assistants with file manipulation, command execution, and code search capabilities.

## Environment Setup

**Using devenv (recommended):**
```bash
devenv shell  # Automatically loads Go, Python, Node, Rust, .NET, ripgrep
```

**Manual setup:**
- Go 1.24.2+
- ripgrep (for code search functionality)
- Set `ANTHROPIC_API_KEY` environment variable

## Building and Running

**Build all commands:**
```bash
go build ./cmd/...
```

**Build specific command:**
```bash
go build -o bin/chat ./cmd/chat
go build -o bin/read ./cmd/read
go build -o bin/api ./cmd/api
```

**Run directly with go run:**
```bash
# Basic chat (no tools)
go run ./cmd/chat/main.go

# File reader agent
go run ./cmd/read/main.go

# Code search agent
go run ./cmd/search/main.go

# Full API agent (all tools)
go run ./cmd/api/main.go
```

**Enable verbose logging:**
```bash
go run ./cmd/chat/main.go --verbose
```

**Install dependencies:**
```bash
go mod tidy
```

## Architecture

### Core Components

**Agent Types** (`internal/agent/`):
- `BasicAgent` - Simple conversational agent without tool support (single loop architecture)
- `ToolAgent` - Advanced agent with tool execution capabilities (nested loop "agentic" architecture)

**Tool System** (`internal/tools/`):
Each tool follows a consistent pattern:
1. Input struct with JSON schema tags (e.g., `ReadFileInput`)
2. Schema generation using `GenerateSchema[T]()`
3. Tool definition with name, description, schema, and function
4. Implementation function taking `json.RawMessage` and returning `(string, error)`

**Available Tools:**
- `read_file` - Read file contents
- `list_files` - List directory contents
- `bash` - Execute shell commands
- `edit` - Edit/create files
- `code_search` - Search code using ripgrep with patterns and filters
- `api_call` - Make API calls

**Common Utilities** (`internal/common/`):
- Command-line flag parsing
- Logging setup (verbose mode)
- Anthropic client initialization
- Input reader creation

### Command Structure

Each command in `cmd/` demonstrates a different agent configuration:
- `cmd/chat` - Basic chat without tools
- `cmd/read` - Chat with file reading
- `cmd/list` - Chat with directory listing
- `cmd/bash` - Chat with command execution
- `cmd/edit` - Chat with file editing
- `cmd/search` - Chat with code search
- `cmd/api` - Full agent with all tools

## Key Design Patterns

**Agentic Loop (ToolAgent):**
The `ToolAgent.Run()` method implements a two-level loop:
1. Outer loop: Get user input
2. Inner loop: Execute tools and feed results back to Claude until no more tool requests

This allows Claude to chain multiple tool calls together to accomplish complex tasks.

**Tool Schema Generation:**
Uses `github.com/invopop/jsonschema` with Go generics to automatically generate JSON schemas from struct definitions. The `jsonschema_description` tag provides field descriptions to Claude.

**Color-Coded Output:**
- Blue (`\u001b[94m`): User input
- Yellow (`\u001b[93m`): Claude responses
- Cyan (`\u001b[96m`): Tool calls
- Green (`\u001b[92m`): Tool results
- Red (`\u001b[91m`): Errors

## Adding New Tools

1. Create input struct with JSON schema tags in `internal/tools/`
2. Generate schema: `var MyToolInputSchema = GenerateSchema[MyToolInput]()`
3. Define tool:
   ```go
   var MyToolDefinition = ToolDefinition{
       Name:        "my_tool",
       Description: "What it does and when to use it",
       InputSchema: MyToolInputSchema,
       Function:    MyToolFunc,
   }
   ```
4. Implement function: `func MyToolFunc(input json.RawMessage) (string, error)`
5. Add to tool list in relevant cmd (e.g., `cmd/api/main.go`)

## Model Configuration

Currently uses `anthropic.ModelClaude3_7SonnetLatest` with 1024 max tokens. To change model or increase token limit, modify the `runInference` method in `internal/agent/basic.go` or `internal/agent/tool.go`.

## Code Search

The `code_search` tool uses ripgrep for fast pattern matching:
- Supports regex patterns
- File type filtering (e.g., `go`, `js`, `py`)
- Case-sensitive/insensitive search
- Results limited to 50 matches by default
- Returns format: `filename:line_number:matching_line`

## Testing

There are currently no test files in this project. When adding tests, follow Go conventions:
- Name test files `*_test.go`
- Place tests next to the code they test
- Run tests with `go test ./...`
