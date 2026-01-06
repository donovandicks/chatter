package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SlashCommand represents a distinct command starting with "/" that performs an action.
type SlashCommand struct {
	Name        string
	Description string
	Execute     func(m *model) (tea.Model, tea.Cmd)
}

// SlashCommandHandler manages the registration, matching, and execution of slash commands.
type SlashCommandHandler struct {
	commands      []SlashCommand
	suggestions   []SlashCommand
	suggestionIdx int
	Active        bool
}

// NewSlashCommandHandler creates a new handler and registers default commands.
func NewSlashCommandHandler() *SlashCommandHandler {
	return &SlashCommandHandler{
		commands: []SlashCommand{
			{
				Name:        "/clear",
				Description: "Reset the conversation",
				Execute: func(m *model) (tea.Model, tea.Cmd) {
					m.messages = []chatMessage{{
						Sender:  "System",
						Content: "Conversation cleared.",
						IsUser:  false,
					}}
					m.agent.ClearHistory()
					m.viewport.SetContent(m.renderMessages())
					return m, nil
				},
			},
		},
		Active: false,
	}
}

// Update checks for slash command input, manages suggestions, and handles selection keys.
func (s *SlashCommandHandler) Update(msg tea.Msg, inputVal string) (bool, string, bool) {
	// Only active if input starts with /
	if !strings.HasPrefix(inputVal, "/") {
		s.Active = false
		return false, "", false
	}

	// Simple autocomplete logic
	query := inputVal
	s.suggestions = []SlashCommand{}
	for _, cmd := range s.commands {
		if strings.HasPrefix(cmd.Name, query) {
			s.suggestions = append(s.suggestions, cmd)
		}
	}

	if len(s.suggestions) > 0 {
		s.Active = true
		// Reset index if out of bounds
		if s.suggestionIdx >= len(s.suggestions) {
			s.suggestionIdx = 0
		}
	} else {
		s.Active = false
	}

	if s.Active {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyUp:
				if s.suggestionIdx > 0 {
					s.suggestionIdx--
				}
				return true, "", false
			case tea.KeyDown:
				if s.suggestionIdx < len(s.suggestions)-1 {
					s.suggestionIdx++
				}
				return true, "", false
			case tea.KeyTab:
				if len(s.suggestions) > 0 {
					selected := s.suggestions[s.suggestionIdx]
					return true, selected.Name, false
				}
			case tea.KeyEnter:
				// If exact match or selected, we let the main loop handle the execution
				// by returning the command string to the input
				if len(s.suggestions) > 0 {
					selected := s.suggestions[s.suggestionIdx]
					s.Active = false
					return true, selected.Name, true
				}
			case tea.KeyEsc:
				s.Active = false
				return true, "", false
			}
		}
	}

	return false, "", false
}

// View renders the slash command suggestion list.
func (s *SlashCommandHandler) View() string {
	if !s.Active {
		return ""
	}

	var views []string
	for i, cmd := range s.suggestions {
		text := cmd.Name + " - " + cmd.Description
		if i == s.suggestionIdx {
			views = append(views, selectedSuggestionStyle.Render("> "+text))
		} else {
			views = append(views, suggestionStyle.Render("  "+text))
		}
	}

	content := strings.Join(views, "\n")
	return "\n" + suggestionContainerStyle.Render(content)
}

// ExecuteCommand finds and executes the command matching the given name.
func (s *SlashCommandHandler) ExecuteCommand(name string, m *model) (bool, tea.Model, tea.Cmd) {
	for _, cmd := range s.commands {
		if cmd.Name == name {
			mod, c := cmd.Execute(m)
			return true, mod, c
		}
	}
	return false, m, nil
}
