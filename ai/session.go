package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/genai"
)

// Session manages the state of a conversation.
type Session struct {
	ID             string
	Agent          *Agent
	History        []*genai.Content
	Stats          SessionStats
	ToolMiddleware ToolMiddleware
}

// NewSession creates a new session for the given agent.
func NewSession(agent *Agent, id string) *Session {
	return &Session{
		ID:             id,
		Agent:          agent,
		History:        make([]*genai.Content, 0),
		ToolMiddleware: agent.Middleware, // Inherit middleware from agent
		Stats: SessionStats{
			SessionID:  id,
			StartTime:  time.Now(),
			ModelUsage: make(map[string]*ModelUsage),
		},
	}
}

// Chat sends a message to the agent and returns the response.
func (s *Session) Chat(ctx context.Context, prompt string) (string, error) {
	// Add user message to history
	s.History = append(s.History, genai.NewContentFromText(prompt, genai.RoleUser))

	// Prepare tools
	var funcDecls []*genai.FunctionDeclaration
	for _, t := range s.Agent.Tools {
		funcDecls = append(funcDecls, t.Decl())
	}
	// Wrap in GenAI Tool struct
	genaiTools := []*genai.Tool{
		{
			FunctionDeclarations: funcDecls,
		},
	}

	opts := GenerateOptions{
		Model:             s.Agent.Model,
		SystemInstruction: s.Agent.SystemPrompt,
		Tools:             genaiTools,
	}

	for {
		start := time.Now()

		response, err := s.Agent.Provider.GenerateContent(ctx, s.History, opts)
		s.Stats.ApiDuration += time.Since(start)

		if err != nil {
			if errors.Is(err, context.Canceled) {
				return "", err
			}
			slog.ErrorContext(ctx, "failed to generate AI response", "error", err)
			return "", err
		}

		s.updateTokenStats(response)

		if len(response.Candidates) == 0 {
			return "", errors.New("no candidates returned")
		}

		candidate := response.Candidates[0]
		s.History = append(s.History, candidate.Content)

		var functionCalls []*genai.FunctionCall
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
		}

		if len(functionCalls) == 0 {
			return response.Text(), nil
		}

		// Handle function calls
		for _, fc := range functionCalls {
			if err := s.handleToolCall(ctx, fc); err != nil {
				// Tool execution error, we should probably add it to history as an error output
				// or just log it. handleToolCall adds the response to history.
			}
		}
	}
}

func (s *Session) updateTokenStats(response *genai.GenerateContentResponse) {
	if response.UsageMetadata != nil {
		inputTokens := int(response.UsageMetadata.PromptTokenCount)
		outputTokens := int(response.UsageMetadata.CandidatesTokenCount)
		totalTokens := int(response.UsageMetadata.TotalTokenCount)

		s.Stats.TotalInputTokens += inputTokens
		s.Stats.TotalOutputTokens += outputTokens
		s.Stats.TotalTokens += totalTokens

		modelName := s.Agent.Model
		if _, ok := s.Stats.ModelUsage[modelName]; !ok {
			s.Stats.ModelUsage[modelName] = &ModelUsage{}
		}
		s.Stats.ModelUsage[modelName].Requests++
		s.Stats.ModelUsage[modelName].InputTokens += inputTokens
		s.Stats.ModelUsage[modelName].OutputTokens += outputTokens
	}
}

func (s *Session) handleToolCall(ctx context.Context, fc *genai.FunctionCall) error {
	toolName := fc.Name
	tool, ok := s.Agent.Tools[toolName]

	var resp map[string]any
	s.Stats.ToolCalls++
	toolStart := time.Now()

	if !ok {
		resp = map[string]any{"error": fmt.Sprintf("tool %q not found", toolName)}
		s.Stats.ToolErrors++
	} else {
		// Default handler
		finalHandler := func(ctx context.Context, args map[string]any) (string, error) {
			return tool.Run(ctx, args)
		}

		var res string
		var err error

		if s.ToolMiddleware != nil {
			res, err = s.ToolMiddleware(ctx, tool, fc.Args, finalHandler)
		} else {
			res, err = finalHandler(ctx, fc.Args)
		}

		if err != nil {
			resp = map[string]any{"error": err.Error()}
			s.Stats.ToolErrors++
		} else {
			resp = map[string]any{"result": res}
		}
	}
	s.Stats.ToolDuration += time.Since(toolStart)

	toolResponseContent := &genai.Content{
		Role: "tool",
		Parts: []*genai.Part{
			{
				FunctionResponse: &genai.FunctionResponse{
					Name:     toolName,
					Response: resp,
				},
			},
		},
	}
	s.History = append(s.History, toolResponseContent)
	return nil
}

// ClearHistory resets the conversation history.
func (s *Session) ClearHistory() {
	s.History = make([]*genai.Content, 0)
}