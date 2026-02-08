// main.go - Chat with read_file + list_files tools
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

	// Define MULTIPLE tools
	// Claude can choose which tool to use based on the user's question
	toolList := []tools.ToolDefinition{
		tools.ReadFileDefinition,  // Read file contents
		tools.ListFilesDefinition, // List directory contents
	}
	if verbose {
		log.Printf("Initialized %d tools", len(toolList))
	}

	// Create and run the agent with multiple tool support
	agent := agent.NewToolAgent(&client, getUserMessage, toolList, verbose, "")
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
