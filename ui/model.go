// Package ui defines the actual UI model.
package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/donovandicks/chatter/ai"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type chatMessage struct {
	Sender  string
	Content string
	IsUser  bool
}

type model struct {
	viewport     viewport.Model
	textInput    textinput.Model
	messages     []chatMessage
	agent        *ai.Agent
	spinner      spinner.Model
	isLoading    bool
	err          error
	autocomplete *Autocomplete
}

type (
	agentResponseMsg string
	errMsg           error
)

func NewModel(agent *ai.Agent) tea.Model {

ti := textinput.New()

ti.Placeholder = "Type a message..."
ti.Focus()
ti.CharLimit = 156
ti.Width = 20

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
		textInput:    ti,
		viewport:     vp,
		messages:     []chatMessage{welcomeMsg},
		agent:        agent,
		spinner:      s,
		isLoading:    false,
		err:          nil,
		autocomplete: NewAutocomplete(),
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	// Delegate to autocomplete first
	handled, newVal, newCursor := m.autocomplete.Update(msg, m.textInput.Value(), m.textInput.Position())
	if handled {
		if newVal != "" {
			m.textInput.SetValue(newVal)
			m.textInput.SetCursor(newCursor)
		}
		return m, nil
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	if m.isLoading {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" && !m.isLoading && !m.autocomplete.Active {
				return m.handleSendMessage()
			}
		}

	case agentResponseMsg:
		m.handleAgentResponse(msg)
		return m, tea.Batch(tiCmd, vpCmd)

	case errMsg:
		m.isLoading = false
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m *model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.viewport.Width = msg.Width
	m.textInput.Width = msg.Width
	m.viewport.Height = msg.Height - 10 // Adjust for input + suggestions space
	m.viewport.SetContent(m.renderMessages())
}

func (m *model) handleSendMessage() (tea.Model, tea.Cmd) {
	userText := m.textInput.Value()
	m.messages = append(m.messages, chatMessage{
		Sender:  "You",
		Content: userText,
		IsUser:  true,
	})
	m.viewport.SetContent(m.renderMessages())
	m.textInput.SetValue("")
	m.viewport.GotoBottom()

	m.isLoading = true
	// Keep the spinner spinning and send the request
	return m, tea.Batch(m.spinner.Tick, sendToAgent(m.agent, userText))
}

func (m *model) handleAgentResponse(msg agentResponseMsg) {
	m.isLoading = false
	m.messages = append(m.messages, chatMessage{
		Sender:  "Gemini",
		Content: string(msg),
		IsUser:  false,
	})
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
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

	if m.isLoading {
		return fmt.Sprintf(
			"%s\n\n%s %s",
			m.viewport.View(),
			m.spinner.View(),
			"Thinking...",
		) + "\n"
	}

	return fmt.Sprintf(
		"%s%s\n\n%s",
		m.viewport.View(),
		suggestionsView,
		m.textInput.View(),
	) + "\n"
}

func sendToAgent(agent *ai.Agent, prompt string) tea.Cmd {
	return func() tea.Msg {
		resp, err := agent.SendMessage(context.Background(), ai.Gemini3Flash, prompt)
		if err != nil {
			return errMsg(err)
		}
		return agentResponseMsg(resp)
	}
}