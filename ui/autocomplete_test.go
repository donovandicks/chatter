package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestAutocomplete_Update(t *testing.T) {
	ac := &Autocomplete{
		allFiles: []string{"main.go", "go.mod", "README.md"},
		Active:   false,
		Lister:   nil,
	}

	// 1. Trigger with "@"
	// Input: "Hello @"
	// Cursor at end
	handled, _, _ := ac.Update(nil, "Hello @", 7)
	assert.False(t, handled, "Should not be handled yet, just active")
	assert.True(t, ac.Active, "Should be active after typing @")
	assert.Len(t, ac.suggestions, 3, "Expected 3 suggestions")

	// 2. Filter with "@main"
	ac.Update(nil, "Hello @main", 11)
	assert.Len(t, ac.suggestions, 1)
	if len(ac.suggestions) > 0 {
		assert.Equal(t, "main.go", ac.suggestions[0], "Expected main.go")
	}

	// 3. Selection
	// Simulate KeyEnter
	ac.suggestionIdx = 0
	handled, newVal, newCursor := ac.Update(tea.KeyMsg{Type: tea.KeyEnter}, "Hello @main", 11)

	assert.True(t, handled, "Expected Enter to be handled")
	// Expect "Hello @main.go " (space added, @ preserved)
	expectedVal := "Hello @main.go "
	assert.Equal(t, expectedVal, newVal, "Unexpected new value")
	assert.Equal(t, 15, newCursor, "Unexpected cursor position")
	assert.False(t, ac.Active, "Should not be active after selection")

	// 4. Alt+Enter should be ignored
	ac.Active = true
	ac.suggestions = []string{"main.go"}
	handled, _, _ = ac.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, "Hello @main", 11)
	assert.False(t, handled, "Alt+Enter should NOT be handled by autocomplete")
}
