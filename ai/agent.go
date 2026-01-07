// Package ai defines all APIs for interacting with AI providers.
package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	_ "embed"

	"github.com/donovandicks/chatter/ai/tools"
	"google.golang.org/genai"
)

//go:embed agent_system.md
var agentSystemPrompt string

//go:embed planner.md
var plannerSystemPrompt string

// GeminiModel represents the specific Gemini model version to use.
type GeminiModel string

// Available Gemini models.
const (
	Gemini3Pro   GeminiModel = "gemini-3-pro-preview"
	Gemini3Flash GeminiModel = "gemini-3-flash-preview"
)

// Agent manages the conversation history and interaction with the Gemini AI model.
type Agent struct {
	client       *genai.Client
	model        GeminiModel
	systemPrompt string
	tools        map[string]tools.FunctionTool
	history      []*genai.Content
	// registry keeps track of available sub-agents for delegation
	registry map[string]*AgentConfig
}

// AgentConfig defines the configuration for an AI agent.
type AgentConfig struct {
	Name         string
	Model        GeminiModel
	SystemPrompt string
	Tools        []tools.FunctionTool
}

// Global registry of agent configurations
var agentRegistry = map[string]AgentConfig{
	"planner": {
		Name:         "planner",
		Model:        Gemini3Pro, // Use smarter model for planning
		SystemPrompt: plannerSystemPrompt,
		Tools: []tools.FunctionTool{
			&tools.ReadFile{},
		},
	},
	// Add more agents here
}

func loadAgentPrompt() (string, error) {
	if agentSystemPrompt == "" {
		return "", errors.New("agent system prompt is empty")
	}
	return agentSystemPrompt, nil
}

// NewAgent creates a new Agent instance with the provided configuration.
func NewAgent(ctx context.Context, config *AgentConfig) (*Agent, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("failed to create AI client"), err)
	}

	agent := &Agent{
		client:       client,
		model:        config.Model,
		systemPrompt: config.SystemPrompt,
		tools:        make(map[string]tools.FunctionTool),
		history:      make([]*genai.Content, 0),
		registry:     make(map[string]*AgentConfig),
	}

	// Register tools
	for _, t := range config.Tools {
		agent.tools[t.Decl().Name] = t
	}

	return agent, nil
}

// NewMainAgent creates the primary agent with the default configuration and delegation capabilities.
func NewMainAgent(ctx context.Context) (*Agent, error) {
	sysPrompt, err := loadAgentPrompt()
	if err != nil {
		return nil, err
	}

	// Define available tools for the main agent (excluding delegate for now)
	mainTools := []tools.FunctionTool{
		&tools.ReadFile{},
	}

	config := &AgentConfig{
		Name:         "main",
		Model:        Gemini3Flash,
		SystemPrompt: sysPrompt,
		Tools:        mainTools,
	}

	agent, err := NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	// Setup delegation
	availableAgents := []string{}
	for name, cfg := range agentRegistry {
		availableAgents = append(availableAgents, name)
		// Copy config to instance registry
		c := cfg
		agent.registry[name] = &c
	}

	delegateTool := &tools.DelegateAgent{
		AvailableAgents: availableAgents,
		Delegator:       agent.delegateTask,
	}
	agent.tools[delegateTool.Decl().Name] = delegateTool

	return agent, nil
}

// delegateTask is the callback used by the DelegateAgent tool.
func (a *Agent) delegateTask(ctx context.Context, agentName string, objective string) (string, error) {
	cfg, ok := a.registry[agentName]
	if !ok {
		return "", fmt.Errorf("agent %q not found", agentName)
	}

	// Create the sub-agent
	subAgent, err := NewAgent(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create sub-agent %q: %w", agentName, err)
	}

	slog.InfoContext(ctx, "Delegating task", "to_agent", agentName, "objective", objective)

	// Send the objective to the sub-agent
	return subAgent.SendMessage(ctx, objective)
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

// ClearHistory resets the conversation history.
func (a *Agent) ClearHistory() {
	a.history = make([]*genai.Content, 0)
}
