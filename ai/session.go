package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"time"

	"google.golang.org/genai"
)

var atFileRegex = regexp.MustCompile(`@(\S+)`)

// Session manages the state of a conversation.
type Session struct {
	ID             string
	Agent          *Agent
	History        []*genai.Content
	Stats          SessionStats
	ToolMiddleware ToolMiddleware
	ReadFiles      map[string]*genai.Content
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
		ReadFiles: make(map[string]*genai.Content),
	}
}

// Chat sends a message to the agent and returns the response.
func (s *Session) Chat(ctx context.Context, prompt string) (string, error) {
	// Add user message to history
	s.History = append(s.History, genai.NewContentFromText(prompt, genai.RoleUser))

	// Inject file content referenced by @mentions
	s.handleFileMentions(prompt)

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

func (s *Session) handleFileMentions(prompt string) {
	matches := atFileRegex.FindAllStringSubmatch(prompt, -1)
	for _, match := range matches {
		path := match[1]
		// Skip if path is empty (though regex avoids empty)
		if path == "" {
			continue
		}

		// Direct read, bypassing permissions for user-initiated @mentions
		content, err := os.ReadFile(path)
		var resp map[string]any
		if err != nil {
			resp = map[string]any{"error": err.Error()}
		} else {
			resp = map[string]any{"result": string(content)}
		}

		// Simulate Model Function Call to satisfy history requirements
		s.History = append(s.History, &genai.Content{
			Role: "model",
			Parts: []*genai.Part{
				{
					FunctionCall: &genai.FunctionCall{
						Name: "read_file",
						Args: map[string]any{"path": path},
					},
				},
			},
		})

		// Simulate Tool Response
		toolResponseContent := &genai.Content{
			Role: "tool",
			Parts: []*genai.Part{
				{
					FunctionResponse: &genai.FunctionResponse{
						Name:     "read_file",
						Response: resp,
					},
				},
			},
		}
		s.History = append(s.History, toolResponseContent)

		// Include in context management tracking
		s.ReadFiles[path] = toolResponseContent
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

	// Context Management Logic
	if toolName == "read_file" {
		if path, ok := fc.Args["path"].(string); ok {
			s.ReadFiles[path] = toolResponseContent
		}
	} else if toolName == "write_file" {
		if path, ok := fc.Args["path"].(string); ok {
			if existingContent, exists := s.ReadFiles[path]; exists {
				// Invalidate the old context
				if len(existingContent.Parts) > 0 && existingContent.Parts[0].FunctionResponse != nil {
					existingContent.Parts[0].FunctionResponse.Response = map[string]any{
						"result": fmt.Sprintf("File %s has been modified by write_file. Content is outdated.", path),
					}
				}
			}
		}
	}

	return nil
}

// ListReadFiles returns a list of files currently in the context.
func (s *Session) ListReadFiles() []string {
	var files []string
	for path := range s.ReadFiles {
		files = append(files, path)
	}
	return files
}

// UpdateFileContext forces a refresh of a file's content in the context.
func (s *Session) UpdateFileContext(ctx context.Context, path string) error {
	contentPtr, exists := s.ReadFiles[path]
	if !exists {
		return fmt.Errorf("file %s is not in the context", path)
	}

	tool, ok := s.Agent.Tools["read_file"]
	if !ok {
		return errors.New("read_file tool not available")
	}

	// We execute the tool directly, bypassing middleware/history append
	// because we are updating existing history in-place.
	newContent, err := tool.Run(ctx, map[string]any{"path": path})
	if err != nil {
		return err
	}

	// Update the existing content in history
	if len(contentPtr.Parts) > 0 && contentPtr.Parts[0].FunctionResponse != nil {
		contentPtr.Parts[0].FunctionResponse.Response = map[string]any{
			"result": newContent,
		}
	}

	return nil
}

// ClearHistory resets the conversation history.
func (s *Session) ClearHistory() {
	s.History = make([]*genai.Content, 0)
	s.ReadFiles = make(map[string]*genai.Content)
}