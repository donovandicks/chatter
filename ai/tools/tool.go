// Package tools defines all custom AI tools.
package tools

import "google.golang.org/genai"

// FunctionTool defines the interface for tools that the AI agent can execute.
type FunctionTool interface {
	Decl() *genai.FunctionDeclaration
	Run(map[string]any) (string, error)
}
