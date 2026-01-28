package ai

import (
	"context"
	"testing"

	"github.com/donovandicks/chatter/ai/tools"
	"github.com/donovandicks/chatter/internal/auth"
	"github.com/donovandicks/chatter/internal/config"
	"github.com/stretchr/testify/assert"
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
	err := session.handleToolCall(ctx, &genai.Part{FunctionCall: readCall, ThoughtSignature: []byte("dummy")})
	assert.NoError(t, err, "handleToolCall read failed")

	// Check if tracked
	_, ok := session.ReadFiles["test.txt"]
	assert.True(t, ok, "File test.txt not tracked in ReadFiles")

	// Verify content in history
	lastMsg := session.History[len(session.History)-1]
	resp := lastMsg.Parts[0].FunctionResponse.Response["result"]
	assert.Equal(t, "original content", resp, "Expected 'original content', got %v", resp)

	// 3. Simulate write_file
	writeCall := &genai.FunctionCall{
		Name: "write_file",
		Args: map[string]any{"path": "test.txt"},
	}
	err = session.handleToolCall(ctx, &genai.Part{FunctionCall: writeCall, ThoughtSignature: []byte("dummy")})
	assert.NoError(t, err, "handleToolCall write failed")

	// Check if old context updated
	readContent := session.ReadFiles["test.txt"]
	updatedResp := readContent.Parts[0].FunctionResponse.Response["result"]
	// Should contain "outdated" or similar
	val, ok := updatedResp.(string)
	assert.True(t, ok, "Expected string response")
	assert.NotEqual(t, "original content", val, "Content not updated after write. Got: %v", updatedResp)

	// 4. Test UpdateFileContext
	// Update mock to return new content
	agent.Tools["read_file"] = MockTool{
		NameVal: "read_file",
		RunFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "new content", nil
		},
	}

	err = session.UpdateFileContext(ctx, "test.txt")
	assert.NoError(t, err, "UpdateFileContext failed")

	// Verify update
	finalResp := session.ReadFiles["test.txt"].Parts[0].FunctionResponse.Response["result"]
	assert.Equal(t, "new content", finalResp, "Expected 'new content', got %v", finalResp)
}

func TestSession_ReadManyFilesContext(t *testing.T) {
	// 1. Setup
	agent := &Agent{
		Tools: make(map[string]tools.FunctionTool),
		Model: "test-model",
	}

	readManyTool := MockTool{
		NameVal: "read_many_files",
		RunFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "--- file1.txt ---\ncontent1\n\n--- dir/file2.txt ---\ncontent2\n", nil
		},
	}
	writeTool := MockTool{
		NameVal: "write_file",
		RunFunc: func(ctx context.Context, args map[string]any) (string, error) {
			return "success", nil
		},
	}

	agent.Tools["read_many_files"] = readManyTool
	agent.Tools["write_file"] = writeTool
	agent.Tools["read_file"] = MockTool{NameVal: "read_file"} // Needed for internal lookup in UpdateFileContext

	session := NewSession(agent, "test-session")

	// 2. Simulate read_many_files
	readManyCall := &genai.FunctionCall{
		Name: "read_many_files",
		Args: map[string]any{"path": "."},
	}

	ctx := context.Background()
	err := session.handleToolCall(ctx, &genai.Part{FunctionCall: readManyCall, ThoughtSignature: []byte("dummy")})
	assert.NoError(t, err, "handleToolCall read_many_files failed")

	// Check if tracked
	_, ok1 := session.ReadFiles["file1.txt"]
	assert.True(t, ok1, "file1.txt not tracked")
	_, ok2 := session.ReadFiles["dir/file2.txt"]
	assert.True(t, ok2, "dir/file2.txt not tracked")

	// 3. Test safety check in UpdateFileContext
	err = session.UpdateFileContext(ctx, "file1.txt")
	assert.Error(t, err, "UpdateFileContext should fail for bulk-read files")
	assert.Contains(t, err.Error(), "bulk operation", "Error message should mention bulk operation")

	// 4. Simulate write_file to invalidate
	writeCall := &genai.FunctionCall{
		Name: "write_file",
		Args: map[string]any{"path": "file1.txt"},
	}
	err = session.handleToolCall(ctx, &genai.Part{FunctionCall: writeCall, ThoughtSignature: []byte("dummy")})
	assert.NoError(t, err, "handleToolCall write failed")

	// Verify both files are invalidated because they share the same contentPtr
	resp1 := session.ReadFiles["file1.txt"].Parts[0].FunctionResponse.Response["result"].(string)
	assert.Contains(t, resp1, "modified", "file1.txt should be invalidated")

	resp2 := session.ReadFiles["dir/file2.txt"].Parts[0].FunctionResponse.Response["result"].(string)
	assert.Contains(t, resp2, "modified", "dir/file2.txt should also be invalidated")
}

func TestSession_PruneHistory(t *testing.T) {
	// Setup Session
	agent := &Agent{
		Model: "test-model",
	}
	session := NewSession(agent, "test-session")

	// Create history with 10 tool outputs
	for i := 0; i < 10; i++ {
		session.History = append(session.History, &genai.Content{
			Role: "tool",
			Parts: []*genai.Part{
				{
					FunctionResponse: &genai.FunctionResponse{
						Name:     "some_tool",
						Response: map[string]any{"result": "verbose output"},
					},
				},
			},
		})
	}

	// Prune (limit ignored in current implementation, relying on hardcoded count)
	// We pass a dummy limit
	session.PruneHistory(config.DefaultAutoPruneTokenLimit)

	// Expect last 5 to be intact, first 5 to be pruned
	prunedCount := 0
	intactCount := 0

	for _, msg := range session.History {
		resp := msg.Parts[0].FunctionResponse.Response["result"]
		if resp == "[Output pruned to save context]" {
			prunedCount++
		} else if resp == "verbose output" {
			intactCount++
		}
	}

	assert.Equal(t, 5, prunedCount, "Expected 5 pruned messages")
	assert.Equal(t, 5, intactCount, "Expected 5 intact messages")
}