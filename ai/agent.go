// Package ai defines all APIs for interacting with AI providers.
package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	_ "embed"

	"github.com/donovandicks/chatter/ai/tools"
	"github.com/donovandicks/chatter/internal/auth"
	"google.golang.org/genai"
)

//go:embed agent_system.md
var agentSystemPrompt string

//go:embed planner.md
var plannerSystemPrompt string

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

// AgentRegistry manages the available agent configurations.
type AgentRegistry struct {
	agents map[string]AgentConfig
}

// SessionStats holds the usage statistics for the agent's session.
type SessionStats struct {
	TotalInputTokens  int
	TotalOutputTokens int
	TotalTokens       int
}

// NewAgentRegistry creates a new, empty agent registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]AgentConfig),
	}
}

// Register adds an agent configuration to the registry.
func (r *AgentRegistry) Register(config AgentConfig) {
	r.agents[config.Name] = config
}

// Get retrieves an agent configuration by name.
func (r *AgentRegistry) Get(name string) (AgentConfig, bool) {
	cfg, ok := r.agents[name]
	return cfg, ok
}

// List returns a list of registered agent names.
func (r *AgentRegistry) List() []string {
	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	return names
}

// Agent manages the conversation history and interaction with the Gemini AI model.
type Agent struct {
	client        *genai.Client
	model         GeminiModel
	systemPrompt  string
	tools         map[string]tools.FunctionTool
	history       []*genai.Content
	registry      *AgentRegistry
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
	return agentSystemPrompt, nil
}

// NewAgent creates a new Agent instance with the provided configuration.
func NewAgent(ctx context.Context, config *AgentConfig, pm *auth.PermissionManager, pr PermissionRequester, registry *AgentRegistry) (*Agent, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to create AI client"), err)
	}

	agent := &Agent{
		client:        client,
		model:         config.Model,
		systemPrompt:  config.SystemPrompt,
		tools:         make(map[string]tools.FunctionTool),
		history:       make([]*genai.Content, 0),
		registry:      registry,
		permManager:   pm,
		permRequester: pr,
	}

	// Register tools
	for _, t := range config.Tools {
		agent.tools[t.Decl().Name] = t
	}

	return agent, nil
}

// NewMainAgent creates the primary agent with the default configuration and delegation capabilities.
func NewMainAgent(ctx context.Context, pm *auth.PermissionManager, pr PermissionRequester) (*Agent, error) {
	sysPrompt, err := loadAgentPrompt()
	if err != nil {
		return nil, err
	}

	// Initialize registry
	registry := NewAgentRegistry()
	
	// Register planner agent
	registry.Register(AgentConfig{
		Name:         "planner",
		Model:        Gemini3Pro,
		SystemPrompt: plannerSystemPrompt,
		Tools: []tools.FunctionTool{
			&tools.ReadFile{},
		},
	})

	// Define available tools for the main agent (excluding delegate for now)
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

	agent, err := NewAgent(ctx, config, pm, pr, registry)
	if err != nil {
		return nil, err
	}

	delegateTool := &tools.DelegateAgent{
		AvailableAgents: registry.List(),
		Delegator:       agent.delegateTask,
	}
	agent.tools[delegateTool.Decl().Name] = delegateTool

	return agent, nil
}

// delegateTask is the callback used by the DelegateAgent tool.
func (a *Agent) delegateTask(ctx context.Context, agentName string, objective string) (string, error) {
	if a.registry == nil {
		return "", errors.New("agent registry not available")
	}

	cfg, ok := a.registry.Get(agentName)
	if !ok {
		return "", fmt.Errorf("agent %q not found", agentName)
	}

	// Create the sub-agent
	// Pass the same registry to sub-agents so they can potentially delegate too (if we wanted to allow that)
	subAgent, err := NewAgent(ctx, &cfg, a.permManager, a.permRequester, a.registry)
	if err != nil {
		return "", fmt.Errorf("failed to create sub-agent %q: %w", agentName, err)
	}

	slog.InfoContext(ctx, "Delegating task", "to_agent", agentName, "objective", objective)

	// Send the objective to the sub-agent
	return subAgent.SendMessage(ctx, objective)
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

	// Check existing permissions
	if a.permManager.Check(action) {
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
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return "", err
			}
			slog.ErrorContext(ctx, "failed to generate AI response", "error", err)
			return "", err
		}

		if response.UsageMetadata != nil {
			a.stats.TotalInputTokens += int(response.UsageMetadata.PromptTokenCount)
			a.stats.TotalOutputTokens += int(response.UsageMetadata.CandidatesTokenCount)
			a.stats.TotalTokens += int(response.UsageMetadata.TotalTokenCount)
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
				// Check permission
				if err := a.checkPermission(ctx, fc.Name, fc.Args); err != nil {
					resp = map[string]any{"error": fmt.Sprintf("permission denied: %v", err)}
				} else {
					res, err := tool.Run(fc.Args)
					if err != nil {
						resp = map[string]any{"error": err.Error()}
					} else {
						resp = map[string]any{"result": res}
					}
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
