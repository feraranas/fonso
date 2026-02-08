// main.go - Chat with read_file tool
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

	// Define the tools available to Claude
	// Currently only has read_file tool
	toolList := []tools.ToolDefinition{tools.ReadFileDefinition}
	if verbose {
		log.Printf("Initialized %d tools", len(toolList))
	}

	// Create and run the agent with tool support
	agent := agent.NewToolAgent(&client, getUserMessage, toolList, verbose, "")
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
