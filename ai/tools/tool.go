// Package tools defines all custom AI tools.
package tools

import "google.golang.org/genai"

type FunctionTool interface {
	Decl() *genai.FunctionDeclaration
	Run(map[string]any) (string, error)
}
