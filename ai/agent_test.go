package ai

import (
	"context"
	"testing"

	"github.com/donovandicks/chatter/ai/tools"
	"github.com/donovandicks/chatter/internal/auth"
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

func TestAgent_checkPermission_WriteFile(t *testing.T) {
	pm := auth.NewPermissionManager()
	pr := &mockPermissionRequester{
		responseLevel: auth.LevelApproveOnce,
	}

	agent := &Agent{
		permManager:   pm,
		permRequester: pr,
		tools: map[string]tools.FunctionTool{
			"write_file": &tools.WriteFile{},
		},
	}

	ctx := context.Background()
	args := map[string]any{
		"path":    "/tmp/test.txt",
		"content": "hello",
	}

	// 1. Check permissions for write_file
	// Should request approval because not yet granted
	err := agent.checkPermission(ctx, "write_file", args)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	expectedAction := auth.Action{
		Type:      "file",
		Operation: "write",
		Target:    "/tmp/test.txt",
	}

	// Clear Diff for comparison as it is generated dynamically
	pr.requestedAction.Diff = ""

	if pr.requestedAction != expectedAction {
		t.Errorf("Expected request for %v, got %v", expectedAction, pr.requestedAction)
	}

	// 2. Grant Session Permission and check again
	// Should STILL request approval because Diff is present (to show the user)
	pm.GrantSession(expectedAction)
	pr.requestedAction = auth.Action{} // Reset

	err = agent.checkPermission(ctx, "write_file", args)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	// We expect the request to still happen because of the Diff
	if pr.requestedAction.Diff == "" {
		t.Error("Expected request with diff even after session grant, but got none")
	}

	// 3. Check for DIFFERENT path in same category
	// Should STILL request approval because of Diff
	args2 := map[string]any{
		"path":    "/tmp/another.txt",
		"content": "bye",
	}
	// Reset requestedAction
	pr.requestedAction = auth.Action{}

	err = agent.checkPermission(ctx, "write_file", args2)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	if pr.requestedAction.Diff == "" {
		t.Error("Expected request with diff for new file even after session grant")
	}
}

func TestAgent_checkPermission_ReadFile(t *testing.T) {
	pm := auth.NewPermissionManager()
	pr := &mockPermissionRequester{
		responseLevel: auth.LevelApproveOnce,
	}

	agent := &Agent{
		permManager:   pm,
		permRequester: pr,
		tools: map[string]tools.FunctionTool{
			"read_file": &tools.ReadFile{},
		},
	}

	ctx := context.Background()
	args := map[string]any{
		"path": "/tmp/read.txt",
	}

	// 1. First check
	err := agent.checkPermission(ctx, "read_file", args)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	expectedAction := auth.Action{
		Type:      "file",
		Operation: "read",
		Target:    "/tmp/read.txt",
	}

	if pr.requestedAction != expectedAction {
		t.Errorf("Expected request for %v, got %v", expectedAction, pr.requestedAction)
	}

	// 2. Grant Session and check again
	// Should NOT request approval because ReadFile has no diff
	pm.GrantSession(expectedAction)
	pr.requestedAction = auth.Action{} // Reset

	err = agent.checkPermission(ctx, "read_file", args)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	if pr.requestedAction != (auth.Action{}) {
		t.Errorf("Expected no request (cached in session), but got %v", pr.requestedAction)
	}
}
