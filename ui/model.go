// Package ui defines the actual UI model.
package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/donovandicks/chatter/ai"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// chatMessage represents a single message in the conversation history.
type chatMessage struct {
	Sender  string
	Content string
	IsUser  bool
}

type model struct {
	viewport      viewport.Model
	textarea      textarea.Model
	messages      []chatMessage
	agent         *ai.Agent
	spinner       spinner.Model
	isLoading     bool
	err           error
	autocomplete  *Autocomplete
	slashCommands *SlashCommandHandler
	cancelRequest context.CancelFunc
}

type (
	agentResponseMsg string
	errMsg           error
)

// NewModel initializes the main application model with the given AI agent.
func NewModel(agent *ai.Agent) tea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.CharLimit = 0 // Unlimited
	ta.SetWidth(20)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	welcomeMsg := chatMessage{
		Sender:  "System",
		Content: "Welcome to Chatter! Type a message and press Enter.",
		IsUser:  false,
	}

	vp := viewport.New(30, 5)
	vp.SetContent(welcomeMsg.Content)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		textarea:      ta,
		viewport:      vp,
		messages:      []chatMessage{welcomeMsg},
		agent:         agent,
		spinner:       s,
		isLoading:     false,
		err:           nil,
		autocomplete:  NewAutocomplete(),
		slashCommands: NewSlashCommandHandler(),
		cancelRequest: nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		taCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	// Check for slash command interactions (autocomplete/filtering)
	if strings.HasPrefix(m.textarea.Value(), "/") {
		handled, newVal, shouldExecute := m.slashCommands.Update(msg, m.textarea.Value())
		if handled {
			if newVal != "" {
				m.textarea.SetValue(newVal)
				m.textarea.SetCursor(len(newVal))
			}
			if shouldExecute {
				return m.tryExecuteCommand()
			}
			return m, nil
		}
	} else {
		// Only check autocomplete if not doing slash command
		cursorIdx := len(m.textarea.Value())
		handled, newVal, _ := m.autocomplete.Update(msg, m.textarea.Value(), cursorIdx)
		if handled {
			if newVal != "" {
				m.textarea.SetValue(newVal)
				m.textarea.SetCursor(len(newVal))
			}
			return m, nil
		}
	}

	m.viewport, vpCmd = m.viewport.Update(msg)
	if m.isLoading {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.handleWindowSize(msg)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEsc:
			if m.isLoading && m.cancelRequest != nil {
				m.cancelRequest()
				m.cancelRequest = nil
				m.isLoading = false
				m.messages = append(m.messages, chatMessage{
					Sender:  "System",
					Content: "Request cancelled.",
					IsUser:  false,
				})
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				return m, nil
			}
		case tea.KeyEnter:
			if msg.Alt {
				m.textarea, taCmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return m, tea.Batch(taCmd, vpCmd, spCmd)
			}
			if m.textarea.Value() != "" && !m.isLoading && !m.autocomplete.Active {
				if strings.HasPrefix(m.textarea.Value(), "/") {
					return m.tryExecuteCommand()
				}
				return m.handleSendMessage()
			}
			return m, nil
		}
	case errMsg:
		if errors.Is(msg, context.Canceled) || strings.Contains(msg.Error(), "context canceled") {
			return m, nil
		}
		m.err = msg
		return m, nil
	case agentResponseMsg:
		m.isLoading = false
		m.cancelRequest = nil // Clear cancel function on success
		m.messages = append(m.messages, chatMessage{
			Sender:  "Gemini",
			Content: string(msg),
			IsUser:  false,
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	m.textarea, taCmd = m.textarea.Update(msg)

	return m, tea.Batch(taCmd, vpCmd, spCmd)
}

// tryExecuteCommand parses and executes the current input as a slash command.
func (m model) tryExecuteCommand() (tea.Model, tea.Cmd) {
	val := m.textarea.Value()
	parts := strings.Fields(val)
	if len(parts) > 0 {
		cmdName := parts[0]
		executed, newModel, cmd := m.slashCommands.ExecuteCommand(cmdName, &m)
		if executed {
			m.textarea.Reset()
			return newModel, cmd
		}
	}
	return m, nil
}

func (m model) handleWindowSize(msg tea.WindowSizeMsg) model {
	m.viewport.Width = msg.Width
	m.textarea.SetWidth(msg.Width)
	m.viewport.Height = msg.Height - 10 // Adjust for input + suggestions space
	m.viewport.SetContent(m.renderMessages())
	return m
}

func (m model) handleSendMessage() (tea.Model, tea.Cmd) {
	userText := m.textarea.Value()
	m.messages = append(m.messages, chatMessage{
		Sender:  "You",
		Content: userText,
		IsUser:  true,
	})
	m.viewport.SetContent(m.renderMessages())
	m.textarea.Reset()
	m.viewport.GotoBottom()

	m.isLoading = true

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelRequest = cancel

	// Keep the spinner spinning and send the request
	return m, tea.Batch(m.spinner.Tick, sendToAgent(ctx, m.agent, userText))
}

func (m model) renderMessages() string {
	if m.viewport.Width == 0 {
		return "Initializing..."
	}

	var rendered []string
	for _, msg := range m.messages {
		rendered = append(rendered, renderMessage(msg.Sender, msg.Content, msg.IsUser, m.viewport.Width))
	}
	return strings.Join(rendered, "\n")
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress Ctrl+C to quit.", m.err)
	}

	suggestionsView := m.autocomplete.View()
	// Overwrite suggestions if slash command is active
	if m.slashCommands.Active {
		suggestionsView = m.slashCommands.View()
	}

	if m.isLoading {
		return fmt.Sprintf(
			"%s\n\n%s %s",
			m.viewport.View(),
			m.spinner.View(),
			"Thinking... (Esc to cancel)",
		) + "\n"
	}

	return fmt.Sprintf(
		"%s%s\n\n%s\n%s",
		m.viewport.View(),
		suggestionsView,
		m.textarea.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Enter to send • Alt+Enter for newline"),
	) + "\n"
}

func sendToAgent(ctx context.Context, agent *ai.Agent, prompt string) tea.Cmd {
	return func() tea.Msg {
		resp, err := agent.SendMessage(ctx, prompt)
		if err != nil {
			return errMsg(err)
		}
		return agentResponseMsg(resp)
	}
}
