// common.go - Shared utilities for chat applications
// This file contains common setup code used by both the basic chat (chat.go)
// and the advanced tool-based chat (read.go)

package common

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

// SetupLogging configures logging based on verbose flag
// In verbose mode: logs to stderr with timestamps and file locations
// In normal mode: minimal logging to stdout
func SetupLogging(verbose bool) {
	if verbose {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	} else {
		log.SetOutput(os.Stdout)
		log.SetFlags(0)
		log.SetPrefix("")
	}
}

// ParseFlags parses command-line flags and returns the verbose setting
func ParseFlags() bool {
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()
	return *verbose
}

// InitClient initializes and returns an Anthropic API client
// The client automatically reads the API key from ANTHROPIC_API_KEY environment variable
func InitClient(verbose bool) anthropic.Client {
	client := anthropic.NewClient()
	if verbose {
		log.Println("Anthropic client initialized")
	}
	return client
}

// CreateInputReader creates a function that reads user input line by line from stdin
// Returns: A function that returns (input string, ok bool) where ok=false means EOF
func CreateInputReader() func() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)
	return func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}
}
