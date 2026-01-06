package tools

import "google.golang.org/genai"

type FunctionTool interface {
	Decl() *genai.FunctionDeclaration
}
