package tools

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// DelegatorFunc is the function signature for delegating a task to another agent.
type DelegatorFunc func(ctx context.Context, agentName string, objective string) (string, error)

// DelegateAgent is a tool that allows an agent to delegate tasks to other specialized agents.
type DelegateAgent struct {
	Delegator       DelegatorFunc
	AvailableAgents []string
}

// Decl returns the function declaration for the delegate_to_agent tool.
func (t *DelegateAgent) Decl() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        "delegate_to_agent",
		Description: "Delegates a complex task or query to a specialized agent. Use this when the request requires specific expertise (e.g., planning, deep architectural analysis).",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"agent_name": {
					Type:        genai.TypeString,
					Description: fmt.Sprintf("The name of the agent to delegate to. Available agents: %s", strings.Join(t.AvailableAgents, ", ")),
					Enum:        t.AvailableAgents,
				},
				"objective": {
					Type:        genai.TypeString,
					Description: "The detailed objective or task description for the sub-agent.",
				},
			},
			Required: []string{"agent_name", "objective"},
		},
	}
}

// Run executes the delegation.
func (t *DelegateAgent) Run(args map[string]any) (string, error) {
	agentName, ok := args["agent_name"].(string)
	if !ok {
		return "", fmt.Errorf("invalid argument: agent_name must be a string")
	}

	objective, ok := args["objective"].(string)
	if !ok {
		return "", fmt.Errorf("invalid argument: objective must be a string")
	}

	if t.Delegator == nil {
		return "", fmt.Errorf("delegator function is not set")
	}

	return t.Delegator(context.Background(), agentName, objective)
}
