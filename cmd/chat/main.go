// main.go - Basic chat command (no tools)
package main

import (
	"context"
	"fmt"

	"github.com/fernankiwi/fonso/internal/agent"
	"github.com/fernankiwi/fonso/internal/common"
)

func main() {
	// Use shared utility functions
	verbose := common.ParseFlags()
	common.SetupLogging(verbose)
	client := common.InitClient(verbose)
	getUserMessage := common.CreateInputReader()

	// Create and run the basic chat agent (no tools)
	agent := agent.NewBasicAgent(&client, getUserMessage, verbose)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
