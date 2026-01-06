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

type model struct {
	viewport        viewport.Model
	textInput       textinput.Model
	messages        []string
	agent           *ai.Agent
	spinner         spinner.Model
	isLoading       bool
	err             error
	allFiles        []string
	suggestions     []string
	suggestionIdx   int
	showSuggestions bool
}

var (
	senderStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	botStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	suggestionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

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

	welcomeMsg := "Welcome to Chatter! Type a message and press Enter."
	vp := viewport.New(30, 5)
	vp.SetContent(welcomeMsg)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	files, _ := listFiles(".")

	return model{
		textInput:       ti,
		viewport:        vp,
		messages:        []string{welcomeMsg},
		agent:           agent,
		spinner:         s,
		isLoading:       false,
		err:             nil,
		allFiles:        files,
		showSuggestions: false,
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

	// Handle autocomplete navigation before text input updates
	if m.showSuggestions {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyUp:
				if m.suggestionIdx > 0 {
					m.suggestionIdx--
				}
				return m, nil
			case tea.KeyDown:
				if m.suggestionIdx < len(m.suggestions)-1 {
					m.suggestionIdx++
				}
				return m, nil
			case tea.KeyEnter, tea.KeyTab:
				if len(m.suggestions) > 0 {
					selected := m.suggestions[m.suggestionIdx]
					cursor := m.textInput.Position()
					value := m.textInput.Value()
					
					// Find the start of the current @mention
					start := strings.LastIndex(value[:cursor], "@")
					if start != -1 {
						newValue := value[:start] + selected + " " + value[cursor:]
						m.textInput.SetValue(newValue)
						m.textInput.SetCursor(start + len(selected) + 1)
						m.showSuggestions = false
						return m, nil
					}
				}
			case tea.KeyEsc:
				m.showSuggestions = false
				return m, nil
			}
		}
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	if m.isLoading {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	// Check for trigger to show/update suggestions
	cursor := m.textInput.Position()
	value := m.textInput.Value()
	lastAt := strings.LastIndex(value[:cursor], "@")
	
	if lastAt != -1 {
		// potential mention, check if there are spaces between @ and cursor
		query := value[lastAt+1 : cursor]
		if !strings.Contains(query, " ") {
			m.suggestions = filterFiles(m.allFiles, query)
			if len(m.suggestions) > 0 {
				m.showSuggestions = true
				// Keep index in bounds if list shrinks
				if m.suggestionIdx >= len(m.suggestions) {
					m.suggestionIdx = 0
				}
			} else {
				m.showSuggestions = false
			}
		} else {
			m.showSuggestions = false
		}
	} else {
		m.showSuggestions = false
	}


	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textInput.Width = msg.Width
		m.viewport.Height = msg.Height - 5
		m.viewport.SetContent(m.renderMessages())

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" && !m.isLoading && !m.showSuggestions {
				userText := m.textInput.Value()
				userMsg := senderStyle.Render("You: ") + userText
				m.messages = append(m.messages, userMsg)
				m.viewport.SetContent(m.renderMessages())
				m.textInput.SetValue("")
				m.viewport.GotoBottom()

				m.isLoading = true
				return m, tea.Batch(tiCmd, vpCmd, m.spinner.Tick, sendToAgent(m.agent, userText))
			}
		}

	case agentResponseMsg:
		m.isLoading = false
		botMsg := botStyle.Render("Gemini: ") + string(msg)
		m.messages = append(m.messages, botMsg)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, tea.Batch(tiCmd, vpCmd)

	case errMsg:
		m.isLoading = false
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m model) renderMessages() string {
	if m.viewport.Width == 0 {
		return strings.Join(m.messages, "\n")
	}

	style := lipgloss.NewStyle().Width(m.viewport.Width)
	var wrapped []string
	for _, msg := range m.messages {
		wrapped = append(wrapped, style.Render(msg))
	}
	return strings.Join(wrapped, "\n")
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress Ctrl+C to quit.", m.err)
	}

	var suggestionsView string
	if m.showSuggestions {
		var views []string
		start := 0
		if m.suggestionIdx > 5 {
			start = m.suggestionIdx - 5
		}
		end := start + 5
		if end > len(m.suggestions) {
			end = len(m.suggestions)
		}

		for i, s := range m.suggestions[start:end] {
			idx := start + i
			if idx == m.suggestionIdx {
				views = append(views, selectedStyle.Render("> "+s))
			} else {
				views = append(views, suggestionStyle.Render("  "+s))
			}
		}
		suggestionsView = "\n" + strings.Join(views, "\n")
	}

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
