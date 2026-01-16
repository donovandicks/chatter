package ai

import "time"

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
