package auth

import (
	"sort"
	"testing"
)

func TestPermissionManager(t *testing.T) {
	pm := NewPermissionManager()

	action1 := Action{Type: "file", Operation: "read", Target: "main.go"}
	action2 := Action{Type: "shell", Operation: "exec", Target: "ls"}
	action3 := Action{Type: "file", Operation: "read", Target: "go.mod"}

	// Test initial state
	if pm.Check(action1) {
		t.Errorf("expected action1 to not be approved initially")
	}

	// Test GrantSession (grants category)
	pm.GrantSession(action1)
	if !pm.Check(action1) {
		t.Errorf("expected action1 to be approved after grant")
	}
	if !pm.Check(action3) {
		t.Errorf("expected action3 to be approved via category grant")
	}
	if pm.Check(action2) {
		t.Errorf("expected action2 to not be approved")
	}

	// Test List
	list := pm.List()
	if len(list) != 1 || list[0] != "Category: file:read" {
		t.Errorf("expected list to contain only category grant, got %v", list)
	}

	// Test Revoke
	pm.Revoke(action1)
	if pm.Check(action1) {
		t.Errorf("expected action1 to be revoked")
	}
	if pm.Check(action3) {
		t.Errorf("expected action3 to be revoked")
	}

	// Test ClearAll
	pm.GrantSession(action1)
	pm.GrantSession(action2)
	pm.ClearAll()
	if pm.Check(action1) || pm.Check(action2) {
		t.Errorf("expected all to be cleared")
	}
	if len(pm.List()) != 0 {
		t.Errorf("expected empty list after clear")
	}
}

func TestActionCategory(t *testing.T) {
	tests := []struct {
		action   Action
		expected string
	}{
		{Action{Type: "file", Operation: "read", Target: "f.txt"}, "file:read"},
		{Action{Type: "shell", Operation: "exec", Target: "ls"}, "shell:exec"},
		{Action{}, "unknown"},
	}

	for _, tt := range tests {
		if got := tt.action.Category(); got != tt.expected {
			t.Errorf("Action.Category() = %v, want %v", got, tt.expected)
		}
	}
}

func TestListSorting(t *testing.T) {
	pm := NewPermissionManager()
	pm.GrantSession(Action{Type: "file", Operation: "read"})
	pm.GrantSession(Action{Type: "shell", Operation: "exec"})

	list := pm.List()
	sort.Strings(list)

	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
	if list[0] != "Category: file:read" || list[1] != "Category: shell:exec" {
		t.Errorf("unexpected list content: %v", list)
	}
}