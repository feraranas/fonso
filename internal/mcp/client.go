// client.go - MCP client that spawns a server subprocess and bridges tools
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fernankiwi/fonso/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClient wraps an MCP client session connected to a subprocess server.
type MCPClient struct {
	session *mcp.ClientSession
	cmd     *exec.Cmd
}

// Connect spawns the MCP server binary as a subprocess and connects to it over stdio.
func Connect(ctx context.Context, serverBinaryPath string) (*MCPClient, error) {
	cmd := exec.Command(serverBinaryPath)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "fonso-agent",
		Version: "v1.0.0",
	}, nil)

	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	return &MCPClient{
		session: session,
		cmd:     cmd,
	}, nil
}

// DiscoverTools lists tools from the MCP server and converts them to ToolDefinitions
// that are indistinguishable from built-in tools.
func (c *MCPClient) DiscoverTools(ctx context.Context) ([]tools.ToolDefinition, error) {
	result, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP tools: %w", err)
	}

	var defs []tools.ToolDefinition
	for _, tool := range result.Tools {
		defs = append(defs, c.bridgeTool(ctx, tool))
	}
	return defs, nil
}

// bridgeTool converts a single MCP Tool into a tools.ToolDefinition.
func (c *MCPClient) bridgeTool(ctx context.Context, tool *mcp.Tool) tools.ToolDefinition {
	// Convert MCP InputSchema to Anthropic ToolInputSchemaParam via JSON round-trip.
	var inputSchema anthropic.ToolInputSchemaParam
	if tool.InputSchema != nil {
		data, err := json.Marshal(tool.InputSchema)
		if err == nil {
			json.Unmarshal(data, &inputSchema)
		}
	}

	toolName := tool.Name
	session := c.session

	return tools.ToolDefinition{
		Name:        tool.Name,
		Description: tool.Description,
		InputSchema: inputSchema,
		Function: func(input json.RawMessage) (string, error) {
			var args map[string]any
			if err := json.Unmarshal(input, &args); err != nil {
				return "", fmt.Errorf("failed to unmarshal tool input: %w", err)
			}

			result, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      toolName,
				Arguments: args,
			})
			if err != nil {
				return "", fmt.Errorf("MCP tool call failed: %w", err)
			}

			if result.IsError {
				return "", fmt.Errorf("MCP tool returned error")
			}

			// Extract text from result content
			var text string
			for _, content := range result.Content {
				if tc, ok := content.(*mcp.TextContent); ok {
					text += tc.Text
				}
			}
			return text, nil
		},
	}
}

// Close shuts down the MCP session and kills the subprocess.
func (c *MCPClient) Close() {
	if c.session != nil {
		c.session.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
}
