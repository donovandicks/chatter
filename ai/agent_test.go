package ai

import (
	"context"
	"testing"

	"github.com/donovandicks/chatter/internal/auth"
	"google.golang.org/genai"
)

type mockPermissionRequester struct {
	requestedAction auth.Action
	responseLevel   auth.PermissionLevel
	responseErr     error
}

func (m *mockPermissionRequester) RequestApproval(ctx context.Context, action auth.Action) (auth.PermissionLevel, error) {
	m.requestedAction = action
	return m.responseLevel, m.responseErr
}

// mockTool implements FunctionTool for testing
type mockTool struct {
	name string
}

func (m mockTool) Decl() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{Name: m.name}
}

func (m mockTool) Run(ctx context.Context, args map[string]any) (string, error) {
	return "ok", nil
}

func (m mockTool) RequestPermission(args map[string]any) auth.Action {
	path, _ := args["path"].(string)
	return auth.Action{
		Type:      "file",
		Operation: "write",
		Target:    path,
		Diff:      "some diff", // simplified
	}
}

func TestPermissionMiddleware(t *testing.T) {
	pm := auth.NewPermissionManager()
	pr := &mockPermissionRequester{
		responseLevel: auth.LevelApproveOnce,
	}

	middleware := NewPermissionMiddleware(pm, pr)
	
	// Mock next handler
	nextHandler := func(ctx context.Context, args map[string]any) (string, error) {
		return "executed", nil
	}

	tool := mockTool{name: "write_file"}
	ctx := context.Background()
	args := map[string]any{
		"path":    "/tmp/test.txt",
		"content": "hello",
	}

	// 1. Check permissions for write_file
	// Should request approval because not yet granted
	_, err := middleware(ctx, tool, args, nextHandler)
	if err != nil {
		t.Fatalf("middleware failed: %v", err)
	}

	expectedAction := auth.Action{
		Type:      "file",
		Operation: "write",
		Target:    "/tmp/test.txt",
		Diff:      "some diff",
	}

	if pr.requestedAction != expectedAction {
		t.Errorf("Expected request for %v, got %v", expectedAction, pr.requestedAction)
	}

	// 2. Grant Session Permission and check again
	// Should STILL request approval because Diff is present (mock tool always returns diff)
	// In the real implementation, WriteFile computes diff. Here mock returns it.
	// Logic: if Diff != "", we ask.
	pm.GrantSession(expectedAction)
	pr.requestedAction = auth.Action{} // Reset

	_, err = middleware(ctx, tool, args, nextHandler)
	if err != nil {
		t.Fatalf("middleware failed: %v", err)
	}

	// We expect the request to still happen because of the Diff
	if pr.requestedAction.Diff == "" {
		t.Error("Expected request with diff even after session grant, but got none")
	}
}
