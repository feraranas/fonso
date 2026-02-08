// main.go - Chat with read_file + list_files + bash tools
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

	// Added bash execution tool
	// This creates a very powerful agent that can:
	// - Read files
	// - List directories
	// - Execute shell commands
	//
	// ⚠️  Use with caution! This can execute ANY command.
	toolList := []tools.ToolDefinition{
		tools.ReadFileDefinition,  // Read file contents
		tools.ListFilesDefinition, // List directory contents
		tools.BashDefinition,      // Execute bash commands
	}
	if verbose {
		log.Printf("Initialized %d tools (including bash execution)", len(toolList))
	}

	// Create and run the agent with command execution capability
	agent := agent.NewToolAgent(&client, getUserMessage, toolList, verbose, "Chat with Claude (use 'ctrl-c' to quit)\n⚠️  Warning: Bash execution is enabled. Claude can run commands on your system!")
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
