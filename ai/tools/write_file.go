package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/genai"
)

// WriteFile is a tool that allows the agent to write content to a file in the local filesystem.
type WriteFile struct{}

// Decl returns the function declaration for the write_file tool.
func (t WriteFile) Decl() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        "write_file",
		Description: "Write content to a file",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"path": {
					Type:        genai.TypeString,
					Description: "The path to the file to write",
				},
				"content": {
					Type:        genai.TypeString,
					Description: "The content to write to the file",
				},
			},
			Required: []string{"path", "content"},
		},
	}
}

// Run executes the write_file tool.
func (t WriteFile) Run(args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("invalid argument: path must be a string")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid argument: content must be a string")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote to %s", path), nil
}
