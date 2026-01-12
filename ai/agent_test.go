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

	if pr.requestedAction != expectedAction {
		t.Errorf("Expected request for %v, got %v", expectedAction, pr.requestedAction)
	}

	// 2. Grant Session Permission and check again
	// Should NOT request approval
	pm.GrantSession(expectedAction)
	pr.requestedAction = auth.Action{} // Reset

	err = agent.checkPermission(ctx, "write_file", args)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	if pr.requestedAction != (auth.Action{}) {
		t.Errorf("Expected no request (cached in session), but got %v", pr.requestedAction)
	}

	// 3. Check for DIFFERENT path in same category
	// Should NOT request approval because category is granted
	args2 := map[string]any{
		"path":    "/tmp/another.txt",
		"content": "bye",
	}
	err = agent.checkPermission(ctx, "write_file", args2)
	if err != nil {
		t.Fatalf("checkPermission failed: %v", err)
	}

	if pr.requestedAction != (auth.Action{}) {
		t.Errorf("Expected no request (category granted), but got %v", pr.requestedAction)
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
}
