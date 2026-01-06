// Package ai defines all APIs for interacting with AI providers.
package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/donovandicks/chatter/ai/tools"
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
	tools        map[string]tools.FunctionTool
	history      []*genai.Content
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
		return nil, errors.Join(fmt.Errorf("failed to create AI client"), err)
	}

	sysPrompt, err := loadAgentPrompt()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to load agent system prompt"), err)
	}

	return &Agent{
		client:       client,
		systemPrompt: sysPrompt,
		tools:        registerTools(),
		history:      make([]*genai.Content, 0),
	}, nil
}

func registerTools() map[string]tools.FunctionTool {
	return map[string]tools.FunctionTool{
		"read_file": new(tools.ReadFile),
	}
}

func (a *Agent) SendMessage(ctx context.Context, model GoogleModel, prompt string) (string, error) {
	var funcDecls []*genai.FunctionDeclaration
	for _, t := range a.tools {
		funcDecls = append(funcDecls, t.Decl())
	}

	// Add user message to history
	a.history = append(a.history, genai.NewContentFromText(prompt, genai.RoleUser))

	// We work with a copy of history for the current turn to handle tool interactions
	// The final model response will be appended to the main history.
	// However, tool calls and responses MUST be part of the history for the model to "see" them in the loop.
	// So we append to a.history as we go.

	for {
		response, err := a.client.Models.GenerateContent(
			ctx,
			string(model),
			a.history,
			&genai.GenerateContentConfig{
				SystemInstruction: genai.NewContentFromText(a.systemPrompt, genai.RoleModel),
				Tools: []*genai.Tool{
					{
						FunctionDeclarations: funcDecls,
					},
				},
			},
		)
		if err != nil {
			slog.ErrorContext(ctx, "failed to generate AI response", "error", err)
			return "", err
		}

		if len(response.Candidates) == 0 {
			return "", errors.New("no candidates returned")
		}

		candidate := response.Candidates[0]
		
		// Add the model's response (which might be a tool call) to history
		a.history = append(a.history, candidate.Content)

		var functionCalls []*genai.FunctionCall
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
		}

		// If no function calls, we are done. Return text.
		if len(functionCalls) == 0 {
			return response.Text(), nil
		}

		// Handle function calls
		for _, fc := range functionCalls {
			tool, ok := a.tools[fc.Name]
			var resp map[string]any
			if !ok {
				resp = map[string]any{"error": fmt.Sprintf("tool %q not found", fc.Name)}
			} else {
				res, err := tool.Run(fc.Args)
				if err != nil {
					resp = map[string]any{"error": err.Error()}
				} else {
					resp = map[string]any{"result": res}
				}
			}

			// Add tool response to history
			toolResponseContent := &genai.Content{
				Role: "tool",
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							Name:     fc.Name,
							Response: resp,
						},
					},
				},
			}
			a.history = append(a.history, toolResponseContent)
		}
	}
}