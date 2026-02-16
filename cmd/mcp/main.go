// main.go - MCP server binary with example tools
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type HelloInput struct {
	Name string `json:"name" jsonschema:"description=The name to greet"`
}

type TimestampInput struct{}

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "fonso-mcp",
		Version: "v1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "hello",
		Description: "Returns a greeting for the given name",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args HelloInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Hello, %s!", args.Name)},
			},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "timestamp",
		Description: "Returns the current UTC timestamp",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args TimestampInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: time.Now().UTC().Format(time.RFC3339)},
			},
		}, nil, nil
	})

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
