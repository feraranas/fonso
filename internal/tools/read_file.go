// read_file.go - Read file tool implementation
package tools

import (
	"encoding/json"
	"log"
	"os"
)

// ReadFileInput defines the parameters for the read_file tool
// The struct tags are used to generate the JSON schema:
// - json:"path" - The parameter name in JSON
// - jsonschema_description - Description shown to Claude about this parameter
type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

// ReadFileInputSchema is the generated JSON schema for ReadFileInput
var ReadFileInputSchema = GenerateSchema[ReadFileInput]()

// ReadFileDefinition is the tool that allows Claude to read files
// When Claude receives a question like "What's in README.md?", it can:
// 1. Decide to call read_file tool
// 2. Provide the path parameter: {"path": "README.md"}
// 3. Receive the file contents as the result
// 4. Use the contents to answer the user's question
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema, // Defines that it takes a "path" parameter
	Function:    ReadFile,            // Points to the actual implementation
}

// ReadFile is the actual implementation of the read_file tool
// It receives JSON input, parses it, reads the file, and returns the contents
func ReadFile(input json.RawMessage) (string, error) {
	// Parse the JSON input to extract the path parameter
	readFileInput := ReadFileInput{}
	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		panic(err)
	}

	log.Printf("Reading file: %s", readFileInput.Path)

	// Read the file from the filesystem
	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		// If file doesn't exist or can't be read, return error
		log.Printf("Failed to read file %s: %v", readFileInput.Path, err)
		return "", err
	}

	log.Printf("Successfully read file %s (%d bytes)", readFileInput.Path, len(content))

	// Return file contents as a string
	return string(content), nil
}
