// main.go - Chat with all tools + api caller + MCP tools
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/fernankiwi/fonso/internal/agent"
	"github.com/fernankiwi/fonso/internal/auth/openai"
	"github.com/fernankiwi/fonso/internal/common"
	mcpclient "github.com/fernankiwi/fonso/internal/mcp"
	"github.com/fernankiwi/fonso/internal/tools"
)

func main() {
	providerName := flag.String("provider", providerAnthropic, "model provider: "+providerAnthropic+" (default) or "+providerOpenAI)
	model := flag.String("model", "", "model slug (default: "+string(agent.DefaultAnthropicModel)+" for anthropic, "+agent.DefaultOpenAIModel+" for "+providerOpenAI+")")
	maxTokens := flag.Int64("max-tokens", 0, "max response tokens for --provider anthropic (default 1024)")
	openAIAuth := flag.String("openai-auth", string(openai.ModeOfficial), "OAuth flavour for "+providerOpenAI+": official (documented Sign in with ChatGPT) or opencode (reuses opencode's client id; unsanctioned)")
	login := flag.Bool("login", false, "sign in with ChatGPT in your browser, save credentials, and exit (needs --provider "+providerOpenAI+")")
	verbose := common.ParseFlags()
	common.SetupLogging(verbose)
	getUserMessage := common.CreateInputReader()

	ctx := context.TODO()

	opts := providerOptions{Name: *providerName, Model: *model, MaxTokens: *maxTokens, OpenAIAuth: *openAIAuth, Verbose: verbose}

	if *login {
		if opts.Name != providerOpenAI {
			fmt.Printf("Error: --login needs --provider %s\n", providerOpenAI)
			os.Exit(2)
		}
		mode, err := openAIMode(opts.OpenAIAuth)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(2)
		}
		loginCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()
		rec, err := openai.Login(loginCtx, openai.Config{Mode: mode})
		if err != nil {
			fmt.Printf("Login failed: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Signed in (%s, mode %s). Credentials saved to %s\n", rec.Email, rec.Mode, openai.DefaultStorePath())
		return
	}

	provider, src, err := newProvider(opts, func() *anthropic.Client {
		c := common.InitClient(verbose)
		return &c
	})
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(2)
	}
	if src != nil {
		// Fail fast (and refresh if needed) before the first prompt.
		if _, _, err := src.Token(ctx); err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
	}

	// Start with all built-in tools
	toolList := append([]tools.ToolDefinition{}, tools.AllTools...)
	if verbose {
		log.Printf("Initialized %d built-in tools", len(toolList))
	}

	// Try to connect to MCP server for additional tools
	mcpClient, mcpTools := connectMCP(ctx, verbose)
	if mcpClient != nil {
		defer mcpClient.Close()
		toolList = append(toolList, mcpTools...)
		if verbose {
			log.Printf("Added %d MCP tools, total: %d", len(mcpTools), len(toolList))
		}
	}

	agent := agent.NewToolAgentWithProvider(provider, getUserMessage, toolList, verbose, "🔍 AI API Caller Agent (use 'ctrl-c' to quit)!\n")
	err = agent.Run(ctx)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

// connectMCP attempts to find and connect to the MCP server binary.
// Returns nil if the server is not available.
func connectMCP(ctx context.Context, verbose bool) (*mcpclient.MCPClient, []tools.ToolDefinition) {
	// Look for the MCP server binary relative to the current executable or in bin/
	mcpPath := findMCPBinary()
	if mcpPath == "" {
		if verbose {
			log.Println("MCP server binary not found, skipping MCP tools")
		}
		return nil, nil
	}

	if verbose {
		log.Printf("Connecting to MCP server: %s", mcpPath)
	}

	client, err := mcpclient.Connect(ctx, mcpPath)
	if err != nil {
		if verbose {
			log.Printf("Failed to connect to MCP server: %v", err)
		}
		return nil, nil
	}

	mcpTools, err := client.DiscoverTools(ctx)
	if err != nil {
		if verbose {
			log.Printf("Failed to discover MCP tools: %v", err)
		}
		client.Close()
		return nil, nil
	}

	if verbose {
		for _, t := range mcpTools {
			log.Printf("Discovered MCP tool: %s", t.Name)
		}
	}

	return client, mcpTools
}

// findMCPBinary searches for the MCP server binary in common locations.
func findMCPBinary() string {
	candidates := []string{
		"./bin/mcp",
		"bin/mcp",
	}

	// Also check relative to executable
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "mcp"))
	}

	for _, path := range candidates {
		if _, err := exec.LookPath(path); err == nil {
			return path
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}
