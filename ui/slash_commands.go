package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// SlashCommand represents a distinct command starting with "/" that performs an action.
type SlashCommand struct {
	Name        string
	Description string
	Execute     func(m *model, args []string) (tea.Model, tea.Cmd)
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
				Execute: func(m *model, args []string) (tea.Model, tea.Cmd) {
					m.messages = []chatMessage{{
						Sender:  "System",
						Content: "Conversation cleared.",
						IsUser:  false,
					}}
					m.session.ClearHistory()
					m.viewport.SetContent(m.renderMessages())
					return m, nil
				},
			},
			{
				Name:        "/compress",
				Description: "Summarize and compact conversation history",
				Execute: func(m *model, args []string) (tea.Model, tea.Cmd) {
					var instructions string
					if len(args) > 0 {
						instructions = strings.Join(args, " ")
					}

					return m, func() tea.Msg {
						// This is a blocking call to the LLM, so run in a Cmd
						err := m.session.Compress(context.Background(), instructions)
						if err != nil {
							return chatMessage{
								Sender:  "System",
								Content: fmt.Sprintf("Compression failed: %v", err),
								IsUser:  false,
							}
						}
						return chatMessage{
							Sender:  "System",
							Content: "Conversation compressed.",
							IsUser:  false,
						}
					}
				},
			},
			{
				Name:        "/context",
				Description: "Manage context (files:list, files:update)",
				Execute: func(m *model, args []string) (tea.Model, tea.Cmd) {
					if len(args) == 0 {
						return addSystemMessage(m, "Usage: /context <files:list|files:update> [args]"), nil
					}

					switch args[0] {
					case "files:list":
						files := m.session.ListReadFiles()
						if len(files) == 0 {
							return addSystemMessage(m, "No files in context."), nil
						}
						var sb strings.Builder
						sb.WriteString("Files in Context:\n")
						for _, f := range files {
							sb.WriteString(fmt.Sprintf("- %s\n", f))
						}
						return addSystemMessage(m, sb.String()), nil
					case "files:update":
						if len(args) < 2 {
							return addSystemMessage(m, "Usage: /context files:update <path>"), nil
						}
						path := args[1]
						// We need a context for the update operation
						// Since Execute is synchronous here, we use Background or a timeout
						// ideally we should do this in a Cmd, but the session update is quick if just reading file
						// However, file I/O should be a Cmd.
						// For simplicity in this CLI structure, we'll do it synchronously or wrap in Cmd if needed.
						// Given the Session method uses context, let's use Background.
						err := m.session.UpdateFileContext(context.Background(), path)
						if err != nil {
							return addSystemMessage(m, fmt.Sprintf("Error updating file context: %v", err)), nil
						}
						return addSystemMessage(m, fmt.Sprintf("Context updated for file: %s", path)), nil
					default:
						return addSystemMessage(m, "Unknown subcommand. Usage: /context <files:list|files:update>"), nil
					}
				},
			},
			{
				Name:        "/perms",
				Description: "Manage permissions (list, clear)",
				Execute: func(m *model, args []string) (tea.Model, tea.Cmd) {
					if len(args) == 0 {
						return addSystemMessage(m, "Usage: /perms <list|clear>"), nil
					}

					pm := m.permRequester.PermManager
					if pm == nil {
						return addSystemMessage(m, "Error: Permission manager not available."), nil
					}

					switch args[0] {
					case "list":
						perms := pm.List()
						if len(perms) == 0 {
							return addSystemMessage(m, "No active session permissions."), nil
						}
						var sb strings.Builder
						sb.WriteString("Active Session Permissions:\n")
						for _, p := range perms {
							sb.WriteString(fmt.Sprintf("- %s\n", p))
						}
						return addSystemMessage(m, sb.String()), nil
					case "clear":
						pm.ClearAll()
						return addSystemMessage(m, "All session permissions cleared."), nil
					default:
						return addSystemMessage(m, "Unknown subcommand. Usage: /perms <list|clear>"), nil
					}
				},
			},
			{
				Name:        "/stats",
				Description: "Show session statistics",
				Execute: func(m *model, args []string) (tea.Model, tea.Cmd) {
					stats := m.session.Stats
					// Pass false to suppress the "Goodbye" message
					output := RenderStats(stats, false)
					return addSystemMessage(m, output), nil
				},
			},
			{
				Name:        "/quit",
				Description: "Exit the session",
				Execute: func(m *model, args []string) (tea.Model, tea.Cmd) {
					return m, tea.Quit
				},
			},
		},
		Active: false,
	}
}

func addSystemMessage(m *model, content string) *model {
	m.messages = append(m.messages, chatMessage{
		Sender:  "System",
		Content: content,
		IsUser:  false,
	})
	m.viewport.SetContent(m.renderMessages())
	return m
}

// Update checks for slash command input, manages suggestions, and handles selection keys.
func (s *SlashCommandHandler) Update(msg tea.Msg, inputVal string) (bool, string, bool) {
	// Only active if input starts with "/"
	if !strings.HasPrefix(inputVal, "/") {
		s.Active = false
		return false, "", false
	}

	// Simple autocomplete logic
	query := strings.Fields(inputVal)[0]
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
				if msg.Alt {
					return false, "", false
				}
				// If exact match or selected, we let the main loop handle the execution
				// by returning the command string to the input
				if len(s.suggestions) > 0 {
					selected := s.suggestions[s.suggestionIdx]
					s.Active = false

					// If the input already contains the full command name, preserve the input (including args).
					parts := strings.Fields(inputVal)
					if len(parts) > 0 && parts[0] == selected.Name {
						return true, inputVal, true
					}

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
func (s *SlashCommandHandler) ExecuteCommand(name string, args []string, m *model) (bool, tea.Model, tea.Cmd) {
	for _, cmd := range s.commands {
		if cmd.Name == name {
			mod, c := cmd.Execute(m, args)
			return true, mod, c
		}
	}
	return false, m, nil
}