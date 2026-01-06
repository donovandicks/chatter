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
		tools: map[string]tools.FunctionTool{
			"read_file": new(tools.ReadFile),
		},
	}, nil
}

func (a *Agent) SendMessage(ctx context.Context, model GoogleModel, prompt string) (string, error) {
	var funcDecls []*genai.FunctionDeclaration
	for _, t := range a.tools {
		funcDecls = append(funcDecls, t.Decl())
	}

	contents := []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}

	for {
		response, err := a.client.Models.GenerateContent(
			ctx,
			string(model),
			contents,
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

		var functionCalls []*genai.FunctionCall
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
		}

		if len(functionCalls) == 0 {
			return response.Text(), nil
		}

		contents = append(contents, candidate.Content)

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

			contents = append(contents, &genai.Content{
				Role: "tool",
				Parts: []*genai.Part{
					{
						FunctionResponse: &genai.FunctionResponse{
							Name:     fc.Name,
							Response: resp,
						},
					},
				},
			})
		}
	}
}
