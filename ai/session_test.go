package ai

import (
	"context"
	"os"
	"testing"

	"github.com/donovandicks/chatter/ai/tools"
	"github.com/donovandicks/chatter/internal/auth"
	"google.golang.org/genai"
)

type MockTool struct {
	NameVal string
	RunFunc func(ctx context.Context, args map[string]any) (string, error)
}

func (m MockTool) Decl() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{Name: m.NameVal}
}

func (m MockTool) Run(ctx context.Context, args map[string]any) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, args)
	}
	return "default result", nil
}

func (m MockTool) RequestPermission(args map[string]any) auth.Action {
	return auth.Action{}
}

type MockProvider struct {
	GenerateContentFunc func(ctx context.Context, history []*genai.Content, opts GenerateOptions) (*genai.GenerateContentResponse, error)
}

func (m *MockProvider) GenerateContent(ctx context.Context, history []*genai.Content, opts GenerateOptions) (*genai.GenerateContentResponse, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, history, opts)
	}
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{
			{
				Content: &genai.Content{
					Parts: []*genai.Part{{Text: "Mock Response"}},
				},
			},
		},
	}, nil
}

func (m *MockProvider) Close() error {
	return nil
}

func TestSession_FileContext(t *testing.T) {
	// 1. Setup
	agent := &Agent{
		Tools: make(map[string]tools.FunctionTool),
		// Model and Provider not strictly needed for handleToolCall
		Model: "test-model",
	}

	readTool := MockTool{
		NameVal: "read_file",
		RunFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "original content", nil
		},
	}
	writeTool := MockTool{
		NameVal: "write_file",
		RunFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "success", nil
		},
	}

	agent.Tools["read_file"] = readTool
	agent.Tools["write_file"] = writeTool

	session := NewSession(agent, "test-session")

	// 2. Simulate read_file
	readCall := &genai.FunctionCall{
		Name: "read_file",
		Args: map[string]any{"path": "test.txt"},
	}

	ctx := context.Background()
	err := session.handleToolCall(ctx, readCall)
	if err != nil {
		t.Fatalf("handleToolCall read failed: %v", err)
	}

	// Check if tracked
	if _, ok := session.ReadFiles["test.txt"]; !ok {
		t.Error("File test.txt not tracked in ReadFiles")
	}

	// Verify content in history
	lastMsg := session.History[len(session.History)-1]
	resp := lastMsg.Parts[0].FunctionResponse.Response["result"]
	if resp != "original content" {
		t.Errorf("Expected 'original content', got %v", resp)
	}

	// 3. Simulate write_file
	writeCall := &genai.FunctionCall{
		Name: "write_file",
		Args: map[string]any{"path": "test.txt"},
	}
	err = session.handleToolCall(ctx, writeCall)
	if err != nil {
		t.Fatalf("handleToolCall write failed: %v", err)
	}

	// Check if old context updated
	readContent := session.ReadFiles["test.txt"]
	updatedResp := readContent.Parts[0].FunctionResponse.Response["result"]
	// Should contain "outdated" or similar
	if val, ok := updatedResp.(string); !ok || val == "original content" {
		t.Errorf("Content not updated after write. Got: %v", updatedResp)
	}

	// 4. Test UpdateFileContext
	// Update mock to return new content
	agent.Tools["read_file"] = MockTool{
		NameVal: "read_file",
		RunFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "new content", nil
		},
	}

	err = session.UpdateFileContext(ctx, "test.txt")
	if err != nil {
		t.Fatalf("UpdateFileContext failed: %v", err)
	}

	// Verify update
	finalResp := session.ReadFiles["test.txt"].Parts[0].FunctionResponse.Response["result"]
	if finalResp != "new content" {
		t.Errorf("Expected 'new content', got %v", finalResp)
	}
}

func TestSession_Chat_AtFile(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "testfile-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	content := "Hello AtFile"
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Setup Agent with Mock Provider
	mockProvider := &MockProvider{
		GenerateContentFunc: func(ctx context.Context, history []*genai.Content, opts GenerateOptions) (*genai.GenerateContentResponse, error) {
			// Check if history contains the file content as a tool response
			// History should be: [UserMsg, ModelCall, ToolResponse]
			if len(history) < 3 {
				t.Errorf("Expected at least 3 messages in history, got %d", len(history))
				return nil, nil
			}
			toolMsg := history[len(history)-1]
			if toolMsg.Role != "tool" {
				t.Errorf("Expected last message to be 'tool', got %s", toolMsg.Role)
			}
			resp := toolMsg.Parts[0].FunctionResponse.Response["result"]
			if resp != "Hello AtFile" {
				t.Errorf("Expected 'Hello AtFile', got %v", resp)
			}
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{
					{Content: &genai.Content{Parts: []*genai.Part{{Text: "OK"}}}},
				},
			}, nil
		},
	}

	agent := &Agent{
		Provider: mockProvider,
		Model:    "test-model",
	}
	session := NewSession(agent, "test-session")

	// Call Chat with @filename
	ctx := context.Background()
	path := tmpFile.Name()
	prompt := "Read @" + path
	_, err = session.Chat(ctx, prompt)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Verify file is tracked in context
	if _, ok := session.ReadFiles[path]; !ok {
		t.Errorf("File %s not tracked in ReadFiles", path)
	}
}
