package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSlashCommandHandler_Update(t *testing.T) {
	sh := NewSlashCommandHandler()

	// 1. Trigger with "/"
	handled, _, _ := sh.Update(nil, "/")
	assert.False(t, handled, "Should not be handled yet")
	assert.True(t, sh.Active, "Should be active")

	// 2. Filter
	// Default commands: /clear
	sh.Update(nil, "/c")
	assert.NotEmpty(t, sh.suggestions, "Expected suggestions for /c")
	if len(sh.suggestions) > 0 {
		assert.Equal(t, "/clear", sh.suggestions[0].Name, "Expected /clear")
	}

	// 3. Select
	// KeyDown (moves to /compress)
	sh.Update(tea.KeyMsg{Type: tea.KeyDown}, "/c")
	// Another KeyDown (moves to /context)
	sh.Update(tea.KeyMsg{Type: tea.KeyDown}, "/c")
	// KeyEnter
	handled, cmdName, execute := sh.Update(tea.KeyMsg{Type: tea.KeyEnter}, "/c")

	assert.True(t, handled, "Expected Enter to be handled")
	assert.Equal(t, "/context", cmdName, "Expected /context")
	assert.True(t, execute, "Expected execute to be true")
	assert.False(t, sh.Active, "Should be inactive after selection")

	// 4. Alt+Enter should be ignored
	sh.Active = true
	sh.suggestions = sh.commands
	handled, _, _ = sh.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true}, "/c")
	assert.False(t, handled, "Alt+Enter should NOT be handled by slash commands")

	// 5. Execute with arguments
	// If the user typed the full command + args, we should return the full input
	input := "/clear all"
	handled, res, exec := sh.Update(tea.KeyMsg{Type: tea.KeyEnter}, input)
	assert.True(t, handled, "Expected Enter to be handled for input with args")
	assert.Equal(t, "/clear all", res, "Expected result to be '/clear all'")
	assert.True(t, exec, "Expected execute=true")
}
