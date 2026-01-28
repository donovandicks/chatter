// Package ai defines all APIs for interacting with AI providers.
package ai

import (
	"context"
	"errors"
	"fmt"
	"os"

	_ "embed"

	"github.com/donovandicks/chatter/ai/tools"
	"github.com/donovandicks/chatter/internal/auth"
)

//go:embed agent_system.md
var agentSystemPrompt string

// AgentConfig defines the configuration for an AI agent.
type AgentConfig struct {
	Name          string
	Model         string
	SystemPrompt  string
	Tools         []tools.FunctionTool
	PermissionMgr *auth.PermissionManager
	PermissionReq PermissionRequester
}

// Agent represents the configuration and capabilities of an AI assistant.
// It is stateless and can be used to create multiple Sessions.
type Agent struct {
	Name         string
	Model        string
	SystemPrompt string
	Tools        map[string]tools.FunctionTool
	Provider     Provider
	Middleware   ToolMiddleware
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
func NewAgent(ctx context.Context, config *AgentConfig) (*Agent, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	// Initialize Provider (Gemini by default for now)
	provider, err := NewGeminiProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create AI provider: %w", err)
	}

	// Build middleware chain
	var mw ToolMiddleware
	if config.PermissionMgr != nil && config.PermissionReq != nil {
		mw = NewPermissionMiddleware(config.PermissionMgr, config.PermissionReq)
	}

	agent := &Agent{
		Name:         config.Name,
		Model:        config.Model,
		SystemPrompt: config.SystemPrompt,
		Tools:        make(map[string]tools.FunctionTool),
		Provider:     provider,
		Middleware:   mw,
	}

	// Register tools
	for _, t := range config.Tools {
		agent.Tools[t.Decl().Name] = t
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
		&tools.ReadManyFiles{},
		&tools.WriteFile{},
	}

	config := &AgentConfig{
		Name:          "main",
		Model:         "gemini-3-flash-preview", // Updated to latest available/preview or keep existing
		SystemPrompt:  sysPrompt,
		Tools:         mainTools,
		PermissionMgr: pm,
		PermissionReq: pr,
	}

	return NewAgent(ctx, config)
}

// NewSession creates a new chat session for this agent.
func (a *Agent) NewSession(id string) *Session {
	return NewSession(a, id)
}

