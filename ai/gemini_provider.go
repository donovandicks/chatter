package ai

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// GeminiProvider implements the Provider interface using Google's GenAI SDK.
type GeminiProvider struct {
	client *genai.Client
}

// NewGeminiProvider creates a new GeminiProvider.
func NewGeminiProvider(ctx context.Context) (*GeminiProvider, error) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}
	return &GeminiProvider{client: client}, nil
}

// GenerateContent sends a request to the Gemini model.
func (p *GeminiProvider) GenerateContent(ctx context.Context, history []*genai.Content, opts GenerateOptions) (*genai.GenerateContentResponse, error) {
	config := &genai.GenerateContentConfig{
		Tools: opts.Tools,
	}

	if opts.SystemInstruction != "" {
		config.SystemInstruction = genai.NewContentFromText(opts.SystemInstruction, genai.RoleModel)
	}

	return p.client.Models.GenerateContent(
		ctx,
		opts.Model,
		history,
		config,
	)
}

// Close closes the underlying client.
func (p *GeminiProvider) Close() error {
	return nil
}