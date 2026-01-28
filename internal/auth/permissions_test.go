package auth

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissionManager(t *testing.T) {
	pm := NewPermissionManager()

	action1 := Action{Type: "file", Operation: "read", Target: "main.go"}
	action2 := Action{Type: "shell", Operation: "exec", Target: "ls"}
	action3 := Action{Type: "file", Operation: "read", Target: "go.mod"}

	// Test initial state
	assert.False(t, pm.Check(action1), "expected action1 to not be approved initially")

	// Test GrantSession (grants category)
	pm.GrantSession(action1)
	assert.True(t, pm.Check(action1), "expected action1 to be approved after grant")
	assert.True(t, pm.Check(action3), "expected action3 to be approved via category grant")
	assert.False(t, pm.Check(action2), "expected action2 to not be approved")

	// Test List
	list := pm.List()
	assert.Len(t, list, 1)
	assert.Equal(t, "Category: file:read", list[0], "expected list to contain only category grant")

	// Test Revoke
	pm.Revoke(action1)
	assert.False(t, pm.Check(action1), "expected action1 to be revoked")
	assert.False(t, pm.Check(action3), "expected action3 to be revoked")

	// Test ClearAll
	pm.GrantSession(action1)
	pm.GrantSession(action2)
	pm.ClearAll()
	assert.False(t, pm.Check(action1), "expected action1 to be cleared")
	assert.False(t, pm.Check(action2), "expected action2 to be cleared")
	assert.Empty(t, pm.List(), "expected empty list after clear")
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
		got := tt.action.Category()
		assert.Equal(t, tt.expected, got, "Action.Category() mismatch")
	}
}

func TestListSorting(t *testing.T) {
	pm := NewPermissionManager()
	pm.GrantSession(Action{Type: "file", Operation: "read"})
	pm.GrantSession(Action{Type: "shell", Operation: "exec"})

	list := pm.List()
	sort.Strings(list)

	assert.Len(t, list, 2, "expected 2 items")
	assert.Equal(t, "Category: file:read", list[0])
	assert.Equal(t, "Category: shell:exec", list[1])
}
