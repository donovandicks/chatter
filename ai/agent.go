// Package ai defines all APIs for interacting with AI providers.
package ai

import (
	"context"
	"log/slog"
	"os"

	"google.golang.org/genai"
)

type GoogleModel string

const (
	Gemini3Pro   GoogleModel = "gemini-3-pro-preview"
	Gemini3Flash GoogleModel = "gemini-3-flash-preview"
)

type Agent struct {
	client       *genai.Client
	systemPrompt string
}

func loadAgentPrompt() (string, error) {
	data, err := os.ReadFile("prompts/agent_system.md")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func NewAgent(ctx context.Context) (*Agent, error) {
	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create AI client", "error", err)
		return nil, err
	}

	sysPrompt, err := loadAgentPrompt()
	if err != nil {
		slog.ErrorContext(ctx, "failed to load agent system prompt", "error", err)
		return nil, err
	}

	return &Agent{
		client:       client,
		systemPrompt: sysPrompt,
	}, nil
}

func (a *Agent) SendMessage(ctx context.Context, model GoogleModel, prompt string) (string, error) {
	response, err := a.client.Models.GenerateContent(
		ctx,
		string(model),
		[]*genai.Content{
			genai.NewContentFromText(prompt, genai.RoleUser),
		},
		&genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(a.systemPrompt, genai.RoleModel),
		},
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate AI response", "error", err)
		return "", err
	}

	return response.Text(), nil
}
