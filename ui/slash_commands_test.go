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
	// KeyDown
	sh.Update(tea.KeyMsg{Type: tea.KeyDown}, "/c")
	// KeyEnter
	handled, cmdName, execute := sh.Update(tea.KeyMsg{Type: tea.KeyEnter}, "/c")

	if !handled {
		t.Error("Expected Enter to be handled")
	}
	if cmdName != "/clear" {
		t.Errorf("Expected /clear, got %s", cmdName)
	}
	if !execute {
		t.Error("Expected execute to be true")
	}
	if sh.Active {
		t.Error("Should be inactive after selection")
	}
}
