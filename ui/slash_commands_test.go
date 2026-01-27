package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlashCommandHandler_Update(t *testing.T) {
	sh := NewSlashCommandHandler()

	// 1. Trigger with "/"
	handled, _, _ := sh.Update(nil, "/")
	if handled {
		t.Error("Should not be handled yet")
	}
	if !sh.Active {
		t.Error("Should be active")
	}

	// 2. Filter
	// Default commands: /clear
	sh.Update(nil, "/c")
	if len(sh.suggestions) == 0 {
		t.Error("Expected suggestions for /c")
	}
	if sh.suggestions[0].Name != "/clear" {
		t.Errorf("Expected /clear, got %s", sh.suggestions[0].Name)
	}

	// 3. Select
	// KeyDown (moves to /compress)
	sh.Update(tea.KeyMsg{Type: tea.KeyDown}, "/c")
	// Another KeyDown (moves to /context)
	sh.Update(tea.KeyMsg{Type: tea.KeyDown}, "/c")
	// KeyEnter
	handled, cmdName, execute := sh.Update(tea.KeyMsg{Type: tea.KeyEnter}, "/c")

	if !handled {
		t.Error("Expected Enter to be handled")
	}
	if cmdName != "/context" {
		t.Errorf("Expected /context, got %s", cmdName)
	}
	if !execute {
		t.Error("Expected execute to be true")
	}
	if sh.Active {
		t.Error("Should be inactive after selection")
	}

	// 4. Alt+Enter should be ignored
	sh.Active = true
	sh.suggestions = sh.commands
	handled, _, _ = sh.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, "/c")
	if handled {
		t.Error("Alt+Enter should NOT be handled by slash commands")
	}

	// 5. Execute with arguments
	// If the user typed the full command + args, we should return the full input
	input := "/clear all"
	handled, res, exec := sh.Update(tea.KeyMsg{Type: tea.KeyEnter}, input)
	if !handled {
		t.Error("Expected Enter to be handled for input with args")
	}
	if res != "/clear all" {
		t.Errorf("Expected result to be '/clear all', got '%s'", res)
	}
	if !exec {
		t.Error("Expected execute=true")
	}
}
