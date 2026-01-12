package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/donovandicks/chatter/internal/auth"
	"google.golang.org/genai"
)

// ReadFile is a tool that allows the agent to read the contents of a file from the local filesystem.
type ReadFile struct{}

// Decl returns the function declaration for the read_file tool, describing its schema to the AI.
func (t ReadFile) Decl() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        "read_file",
		Description: "Read the content of a file",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"path": {
					Type:        genai.TypeString,
					Description: "The path to the file to read",
				},
			},
			Required: []string{"path"},
		},
	}
}

// RequestPermission returns the permission action required for this tool.
func (t ReadFile) RequestPermission(args map[string]any) auth.Action {
	path, _ := args["path"].(string)
	return auth.Action{
		Type:      "file",
		Operation: "read",
		Target:    path,
	}
}

// Run executes the read_file tool with the provided arguments.
func (t ReadFile) Run(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("invalid argument: path must be a string")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(data), nil
}
