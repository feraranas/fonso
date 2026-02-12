// main.go - Chat with all tools + api caller
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/fernankiwi/fonso/internal/agent"
	"github.com/fernankiwi/fonso/internal/common"
	"github.com/fernankiwi/fonso/internal/tools"
)

func main() {
	// Use shared utility functions
	verbose := common.ParseFlags()
	common.SetupLogging(verbose)
	client := common.InitClient(verbose)
	getUserMessage := common.CreateInputReader()

	// Added code search tool
	// This creates a "smart" coding agent that can:
	// - Read files (explore code)
	// - List directories (understand structure)
	// - Execute commands (run tests, build, etc.)
	// - Edit/create files
	// - Search code patterns (find definitions, usage) ← NEW!
	// -  API caller
	// This is closer to how real AI coding assistants work!
	toolList := []tools.ToolDefinition{
		tools.ReadFileDefinition,   // Read file contents
		tools.ListFilesDefinition,  // List directory contents
		tools.BashDefinition,       // Execute bash commands
		tools.EditFileDefinition,   // Edit/create files
		tools.CodeSearchDefinition, // Search code patterns (NEW!)
		tools.ApiCallDefinition,    // API caller
	}
	if verbose {
		log.Printf("Initialized %d tools (including API Caller)", len(toolList))
	}

	// Create and run the code search agent
	agent := agent.NewToolAgent(&client, getUserMessage, toolList, verbose, "🔍 AI API Caller Agent (use 'ctrl-c' to quit)!\n")
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
