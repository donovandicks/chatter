package auth

import (
	"reflect"
	"sort"
	"testing"
)

func TestPermissionManager(t *testing.T) {
	pm := NewPermissionManager()

	action1 := Action{Type: "file", Operation: "read", Target: "main.go"}
	action2 := Action{Type: "shell", Operation: "exec", Target: "ls"}

	// Test initial state
	if pm.Check(action1) {
		t.Errorf("expected action1 to not be approved initially")
	}

	// Test GrantSession
	pm.GrantSession(action1)
	if !pm.Check(action1) {
		t.Errorf("expected action1 to be approved after grant")
	}
	if pm.Check(action2) {
		t.Errorf("expected action2 to not be approved")
	}

	// Test List
	list := pm.List()
	if len(list) != 1 || list[0] != action1 {
		t.Errorf("expected list to contain only action1, got %v", list)
	}

	// Test GrantSession second action
	pm.GrantSession(action2)
	if !pm.Check(action2) {
		t.Errorf("expected action2 to be approved after grant")
	}

	list = pm.List()
	if len(list) != 2 {
		t.Errorf("expected list to contain 2 actions")
	}

	// Sort list for comparison since map iteration is random
	sort.Slice(list, func(i, j int) bool {
		return list[i].Target < list[j].Target
	})
	expected := []Action{action2, action1} // ls < main.go
	if !reflect.DeepEqual(list, expected) {
		t.Errorf("expected list %v, got %v", expected, list)
	}

	// Test Revoke
	pm.Revoke(action1)
	if pm.Check(action1) {
		t.Errorf("expected action1 to be revoked")
	}
	if !pm.Check(action2) {
		t.Errorf("expected action2 to remain approved")
	}

	// Test ClearAll
	pm.ClearAll()
	if pm.Check(action2) {
		t.Errorf("expected action2 to be cleared")
	}
	if len(pm.List()) != 0 {
		t.Errorf("expected empty list after clear")
	}
}
