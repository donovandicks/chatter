package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/donovandicks/chatter/internal/auth"
	"google.golang.org/genai"
)

// ReadManyFiles is a tool that allows the agent to read the contents of all files in a directory.
type ReadManyFiles struct{}

// Decl returns the function declaration for the read_many_files tool.
func (t ReadManyFiles) Decl() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        "read_many_files",
		Description: "Read the content of all files in a directory",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"path": {
					Type:        genai.TypeString,
					Description: "The path to the directory to read",
				},
			},
			Required: []string{"path"},
		},
	}
}

// RequestPermission returns the permission action required for this tool.
func (t ReadManyFiles) RequestPermission(args map[string]any) auth.Action {
	path, _ := args["path"].(string)
	return auth.Action{
		Type:      "file",
		Operation: "read",
		Target:    path,
	}
}

// Run executes the read_many_files tool.
func (t ReadManyFiles) Run(ctx context.Context, args map[string]any) (string, error) {
	dirPath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("invalid argument: path must be a string")
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var sb strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		sb.WriteString(fmt.Sprintf("--- %s ---\n", filePath))
		sb.Write(data)
		sb.WriteString("\n\n")
	}

	if sb.Len() == 0 {
		return "(No files found in directory)", nil
	}

	return sb.String(), nil
}
