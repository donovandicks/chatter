// Package ai defines all APIs for interacting with AI providers.
package ai

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "embed"

	"github.com/donovandicks/chatter/ai/tools"
	"github.com/donovandicks/chatter/internal/auth"
	"google.golang.org/genai"
)

//go:embed agent_system.md
var agentSystemPrompt string

// PermissionRequester defines the interface for requesting user approval for actions.
type PermissionRequester interface {
	RequestApproval(ctx context.Context, action auth.Action) (auth.PermissionLevel, error)
}

// GeminiModel represents the specific Gemini model version to use.
type GeminiModel string

// Available Gemini models.
const (
	Gemini3Pro   GeminiModel = "gemini-3-pro-preview"
	Gemini3Flash GeminiModel = "gemini-3-flash-preview"
)

// ModelUsage tracks usage statistics for a specific model.
type ModelUsage struct {
	Requests     int
	InputTokens  int
	OutputTokens int
}

// SessionStats holds the usage statistics for the agent's session.
type SessionStats struct {
	SessionID         string
	StartTime         time.Time
	TotalInputTokens  int
	TotalOutputTokens int
	TotalTokens       int
	ApiDuration       time.Duration
	ToolDuration      time.Duration
	ToolCalls         int
	ToolErrors        int
	ModelUsage        map[string]*ModelUsage
}

// Agent manages the conversation history and interaction with the Gemini AI model.
type Agent struct {
	client        *genai.Client
	model         GeminiModel
	systemPrompt  string
	tools         map[string]tools.FunctionTool
	history       []*genai.Content
	permManager   *auth.PermissionManager
	permRequester PermissionRequester
	stats         SessionStats
}

// AgentConfig defines the configuration for an AI agent.
type AgentConfig struct {
	Name         string
	Model        GeminiModel
	SystemPrompt string
	Tools        []tools.FunctionTool
}

func loadAgentPrompt() (string, error) {
	if agentSystemPrompt == "" {
		return "", errors.New("agent system prompt is empty")
	}

	prompt := agentSystemPrompt
	if data, err := os.ReadFile("AGENTS.md"); err == nil {
		prompt += "\n\n" + string(data)
	}

	return prompt, nil
}

// NewAgent creates a new Agent instance with the provided configuration.
func NewAgent(ctx context.Context, config *AgentConfig, pm *auth.PermissionManager, pr PermissionRequester) (*Agent, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to create AI client"), err)
	}

	// Generate a random session ID
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}
	sessionID := hex.EncodeToString(b)

	agent := &Agent{
		client:        client,
		model:         config.Model,
		systemPrompt:  config.SystemPrompt,
		tools:         make(map[string]tools.FunctionTool),
		history:       make([]*genai.Content, 0),
		permManager:   pm,
		permRequester: pr,
		stats: SessionStats{
			SessionID:  sessionID,
			StartTime:  time.Now(),
			ModelUsage: make(map[string]*ModelUsage),
		},
	}

	// Register tools
	for _, t := range config.Tools {
		agent.tools[t.Decl().Name] = t
	}

	return agent, nil
}

// NewMainAgent creates the primary agent with the default configuration.
func NewMainAgent(ctx context.Context, pm *auth.PermissionManager, pr PermissionRequester) (*Agent, error) {
	sysPrompt, err := loadAgentPrompt()
	if err != nil {
		return nil, err
	}

	// Define available tools for the main agent
	mainTools := []tools.FunctionTool{
		&tools.ReadFile{},
		&tools.WriteFile{},
	}

	config := &AgentConfig{
		Name:         "main",
		Model:        Gemini3Flash,
		SystemPrompt: sysPrompt,
		Tools:        mainTools,
	}

	return NewAgent(ctx, config, pm, pr)
}

func (a *Agent) checkPermission(ctx context.Context, toolName string, args map[string]any) error {
	// Skip permission check if components are missing (e.g. tests)
	if a.permManager == nil || a.permRequester == nil {
		return nil
	}

	tool, ok := a.tools[toolName]
	if !ok {
		return fmt.Errorf("tool %q not found", toolName)
	}

	// Get the required action from the tool itself
	action := tool.RequestPermission(args)

	// Check existing permissions.
	// We always call RequestApproval if there is a diff to ensure it's displayed to the user
	// (either in the chat log or as a blocking request).
	if action.Diff == "" && a.permManager.Check(action) {
		return nil
	}

	// Request approval
	level, err := a.permRequester.RequestApproval(ctx, action)
	if err != nil {
		return err
	}

	switch level {
	case auth.LevelReject:
		return fmt.Errorf("permission denied by user")
	case auth.LevelApproveOnce:
		return nil
	case auth.LevelApproveSession:
		a.permManager.GrantSession(action)
		return nil
	default:
		return fmt.Errorf("unknown permission level")
	}
}

// SendMessage sends a user prompt to the AI model, handles any tool calls, and returns the final text response.
func (a *Agent) SendMessage(ctx context.Context, prompt string) (string, error) {
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
		start := time.Now()
		response, err := a.client.Models.GenerateContent(
			ctx,
			string(a.model),
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
		a.stats.ApiDuration += time.Since(start)

		if err != nil {
			if errors.Is(err, context.Canceled) {
				return "", err
			}
			slog.ErrorContext(ctx, "failed to generate AI response", "error", err)
			return "", err
		}

		if response.UsageMetadata != nil {
			inputTokens := int(response.UsageMetadata.PromptTokenCount)
			outputTokens := int(response.UsageMetadata.CandidatesTokenCount)
			totalTokens := int(response.UsageMetadata.TotalTokenCount)

			a.stats.TotalInputTokens += inputTokens
			a.stats.TotalOutputTokens += outputTokens
			a.stats.TotalTokens += totalTokens

			// Update per-model stats
			modelName := string(a.model)
			if _, ok := a.stats.ModelUsage[modelName]; !ok {
				a.stats.ModelUsage[modelName] = &ModelUsage{}
			}
			a.stats.ModelUsage[modelName].Requests++
			a.stats.ModelUsage[modelName].InputTokens += inputTokens
			a.stats.ModelUsage[modelName].OutputTokens += outputTokens
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

			a.stats.ToolCalls++
			toolStart := time.Now()

			if !ok {
				resp = map[string]any{"error": fmt.Sprintf("tool %q not found", fc.Name)}
				a.stats.ToolErrors++
			} else {
				// Check permission
				if err := a.checkPermission(ctx, fc.Name, fc.Args); err != nil {
					resp = map[string]any{"error": fmt.Sprintf("permission denied: %v", err)}
					a.stats.ToolErrors++
				} else {
					res, err := tool.Run(ctx, fc.Args)
					if err != nil {
						resp = map[string]any{"error": err.Error()}
						a.stats.ToolErrors++
					} else {
						resp = map[string]any{"result": res}
					}
				}
			}
			a.stats.ToolDuration += time.Since(toolStart)

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

// ClearHistory resets the conversation history.
func (a *Agent) ClearHistory() {
	a.history = make([]*genai.Content, 0)
}

// GetPermissionManager returns the agent's permission manager.
func (a *Agent) GetPermissionManager() *auth.PermissionManager {
	return a.permManager
}

// GetStats returns the accumulated usage statistics for the session.
func (a *Agent) GetStats() SessionStats {
	return a.stats
}
