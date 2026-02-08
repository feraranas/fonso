// main.go - Chat with read_file + list_files + bash + edit_file tools
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

	// Added file editing tool
	// This creates the MOST POWERFUL agent in our examples:
	// - Read files (explore code)
	// - List directories (understand structure)
	// - Execute commands (run tests, build, etc.)
	// - Edit files (modify code!) ← NEW!
	//
	// ⚠️⚠️ EXTREME DANGER! Can modify your code! ⚠️⚠️
	toolList := []tools.ToolDefinition{
		tools.ReadFileDefinition,  // Read file contents
		tools.ListFilesDefinition, // List directory contents
		tools.BashDefinition,      // Execute bash commands
		tools.EditFileDefinition,  // Edit/create files
	}
	if verbose {
		log.Printf("⚠️  Initialized %d tools (including FILE EDITING)", len(toolList))
	}

	// Create and run the coding agent
	agent := agent.NewToolAgent(&client, getUserMessage, toolList, verbose, "🤖 AI Coding Agent (use 'ctrl-c' to quit)\n⚠️⚠️ WARNING: This agent can MODIFY FILES on your system! ⚠️⚠️\n🛡️  Only use in a safe, version-controlled environment!\n")
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
