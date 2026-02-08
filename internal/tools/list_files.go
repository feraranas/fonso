// list_files.go - List files tool implementation
package tools

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ListFilesInput defines parameters for list_files tool
// Note: Path is optional (omitempty) - defaults to current directory
type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

// ListFilesInputSchema is the generated JSON schema for ListFilesInput
var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

// ListFilesDefinition allows Claude to list directory contents
var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory. Use this to explore the project structure or find files.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

// ListFiles implements the list_files tool
// This walks the directory tree and returns a JSON array of files/directories
func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		panic(err)
	}

	// Default to current directory if no path specified
	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	log.Printf("Listing files in directory: %s", dir)

	var files []string

	// Walk the directory tree
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from the base directory
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		// Skip certain directories
		// Add more patterns here if needed (e.g., node_modules, .git)
		if info.IsDir() && (relPath == ".devenv" || strings.HasPrefix(relPath, ".devenv/")) {
			return filepath.SkipDir
		}

		// Add file/directory to list (skip the base directory itself)
		if relPath != "." {
			if info.IsDir() {
				// Add trailing slash to indicate it's a directory
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to list files in %s: %v", dir, err)
		return "", err
	}

	log.Printf("Successfully listed %d items in %s", len(files), dir)

	// Return as JSON array for easy parsing by Claude
	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
