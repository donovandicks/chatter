package tools

import (
	"fmt"
	"os"

	"google.golang.org/genai"
)

type ReadFile struct{}

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

func (t ReadFile) Run(args map[string]any) (string, error) {
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
