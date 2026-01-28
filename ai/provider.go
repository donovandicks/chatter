package ai

import (
	"context"

	"google.golang.org/genai"
)

// GenerateOptions contains options for content generation.
type GenerateOptions struct {
	Model             string
	SystemInstruction string
	Tools             []*genai.Tool
}

// Provider defines the interface for an AI model provider.
type Provider interface {
	// GenerateContent sends a request to the AI model.
	GenerateContent(ctx context.Context, history []*genai.Content, opts GenerateOptions) (*genai.GenerateContentResponse, error)

	// Close cleans up any resources used by the provider.
	Close() error
}
