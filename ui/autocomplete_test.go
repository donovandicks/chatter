package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	if handled {
		t.Error("Should not be handled yet, just active")
	}
	if !ac.Active {
		t.Error("Should be active after typing @")
	}
	if len(ac.suggestions) != 3 {
		t.Errorf("Expected 3 suggestions, got %d", len(ac.suggestions))
	}

	// 2. Filter with "@main"
	ac.Update(nil, "Hello @main", 11)
	if len(ac.suggestions) != 1 || ac.suggestions[0] != "main.go" {
		t.Errorf("Expected main.go, got %v", ac.suggestions)
	}

	// 3. Selection
	// Simulate KeyEnter
	ac.suggestionIdx = 0
	handled, newVal, newCursor := ac.Update(tea.KeyMsg{Type: tea.KeyEnter}, "Hello @main", 11)

	if !handled {
		t.Error("Expected Enter to be handled")
	}
		// Expect "Hello @main.go " (space added, @ preserved)
	expectedVal := "Hello @main.go "
	if newVal != expectedVal {
		t.Errorf("Expected %q, got %q", expectedVal, newVal)
	}
	if newCursor != 15 {
		t.Errorf("Expected cursor at 15, got %d", newCursor)
	}
	if ac.Active {
		t.Error("Should not be active after selection")
	}

	// 4. Alt+Enter should be ignored
	ac.Active = true
	ac.suggestions = []string{"main.go"}
	handled, _, _ = ac.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, "Hello @main", 11)
	if handled {
		t.Error("Alt+Enter should NOT be handled by autocomplete")
	}
}
