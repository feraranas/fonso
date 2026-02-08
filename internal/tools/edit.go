// edit.go - File editing tool implementation
package tools

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
)

// EditFileInput defines parameters for edit_file tool
type EditFileInput struct {
	Path   string `json:"path" jsonschema_description:"The path to the file to edit or create."`
	OldStr string `json:"old_str" jsonschema_description:"Text to search for - must match exactly and must only have one match. Can be empty to create a new file."`
	NewStr string `json:"new_str" jsonschema_description:"Text to replace old_str with, or content for a new file."`
}

// EditFileInputSchema is the generated JSON schema for EditFileInput
var EditFileInputSchema = GenerateSchema[EditFileInput]()

// EditFileDefinition allows Claude to edit/create files
//
// How it works:
// - Searches for `old_str` in the file
// - Replaces it with `new_str`
// - Requires EXACTLY one match (for safety)
// - Can create new files if old_str is empty
//
// Safety features:
// - old_str must be unique (prevents accidental multiple changes)
// - old_str and new_str must be different
// - Creates directories as needed
// - Returns clear error messages
var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `Make edits to a text file.

Replaces 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.

The 'old_str' must appear EXACTLY ONCE in the file for safety (to prevent accidental multiple replacements).

If the file doesn't exist and old_str is empty, a new file will be created with new_str as content.`,
	InputSchema: EditFileInputSchema,
	Function:    EditFile,
}

// EditFile implements the edit_file tool - modify files!
//
// How it works:
// 1. Parse the edit request (path, old_str, new_str)
// 2. Validate inputs (old_str != new_str, path not empty)
// 3. Read the file (or create if it doesn't exist)
// 4. Search for old_str - must find EXACTLY one match
// 5. Replace old_str with new_str
// 6. Write the modified content back to the file
func EditFile(input json.RawMessage) (string, error) {
	editFileInput := EditFileInput{}
	err := json.Unmarshal(input, &editFileInput)
	if err != nil {
		return "", err
	}

	// Validate inputs
	if editFileInput.Path == "" || editFileInput.OldStr == editFileInput.NewStr {
		log.Printf("EditFile failed: invalid input parameters")
		return "", fmt.Errorf("invalid input parameters: path must not be empty and old_str must differ from new_str")
	}

	log.Printf("Editing file: %s (replacing %d chars with %d chars)", editFileInput.Path, len(editFileInput.OldStr), len(editFileInput.NewStr))

	// Try to read the file
	content, err := os.ReadFile(editFileInput.Path)
	if err != nil {
		// If file doesn't exist and old_str is empty, create a new file
		if os.IsNotExist(err) && editFileInput.OldStr == "" {
			log.Printf("File does not exist, creating new file: %s", editFileInput.Path)
			return createNewFile(editFileInput.Path, editFileInput.NewStr)
		}
		log.Printf("Failed to read file %s: %v", editFileInput.Path, err)
		return "", err
	}

	oldContent := string(content)

	// Special case: if old_str is empty, we're appending to the file
	var newContent string
	if editFileInput.OldStr == "" {
		newContent = oldContent + editFileInput.NewStr
		log.Printf("Appending to file: %s", editFileInput.Path)
	} else {
		// Count occurrences first to ensure we have exactly one match
		count := strings.Count(oldContent, editFileInput.OldStr)
		if count == 0 {
			log.Printf("EditFile failed: old_str not found in file %s", editFileInput.Path)
			return "", fmt.Errorf("old_str not found in file. Make sure to read the file first to see its exact content")
		}
		if count > 1 {
			log.Printf("EditFile failed: old_str found %d times in file %s, must be unique", count, editFileInput.Path)
			return "", fmt.Errorf("old_str found %d times in file, must be unique. Include more context in old_str to make it unique", count)
		}

		// Perform the replacement
		newContent = strings.Replace(oldContent, editFileInput.OldStr, editFileInput.NewStr, 1)
	}

	// Write the modified content back to the file
	err = os.WriteFile(editFileInput.Path, []byte(newContent), 0644)
	if err != nil {
		log.Printf("Failed to write file %s: %v", editFileInput.Path, err)
		return "", err
	}

	log.Printf("Successfully edited file %s", editFileInput.Path)
	return "OK", nil
}

// createNewFile creates a new file with the given content
// Also creates any necessary parent directories
func createNewFile(filePath, content string) (string, error) {
	log.Printf("Creating new file: %s (%d bytes)", filePath, len(content))

	// Create parent directories if needed
	dir := path.Dir(filePath)
	if dir != "." {
		log.Printf("Creating directory: %s", dir)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("Failed to create directory %s: %v", dir, err)
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Create the file
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Printf("Failed to create file %s: %v", filePath, err)
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	log.Printf("Successfully created file %s", filePath)
	return fmt.Sprintf("Successfully created file %s", filePath), nil
}
