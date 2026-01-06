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
	viewport        viewport.Model
	textInput       textinput.Model
	messages        []chatMessage
	agent           *ai.Agent
	spinner         spinner.Model
	isLoading       bool
	err             error
	allFiles        []string
	suggestions     []string
	suggestionIdx   int
	showSuggestions bool
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
	
	// We'll set content in the first Update/View or initialization if possible, 
	// but viewport needs width to render correctly. 
	// For now, simple text or wait for WindowSizeMsg.
	vp.SetContent(welcomeMsg.Content) 

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	files, _ := listFiles(".")

	return model{
		textInput:       ti,
		viewport:        vp,
		messages:        []chatMessage{welcomeMsg},
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
		m.viewport.Height = msg.Height - 10 // Adjust for input + suggestions space
		m.viewport.SetContent(m.renderMessages())

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" && !m.isLoading && !m.showSuggestions {
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
				return m, tea.Batch(tiCmd, vpCmd, m.spinner.Tick, sendToAgent(m.agent, userText))
			}
		}

	case agentResponseMsg:
		m.isLoading = false
		m.messages = append(m.messages, chatMessage{
			Sender:  "Gemini",
			Content: string(msg),
			IsUser:  false,
		})
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

	var suggestionsView string
	if m.showSuggestions {
		var views []string
		
		// Window size
		windowSize := 5
		
		// Ensure the selected index is visible
		start := 0
		if m.suggestionIdx >= windowSize {
			start = m.suggestionIdx - windowSize + 1
		}
		
		end := start + windowSize
		if end > len(m.suggestions) {
			end = len(m.suggestions)
			// Adjust start if we hit the bottom but have space at the top
			start = end - windowSize
			if start < 0 {
				start = 0
			}
		}

		for i, s := range m.suggestions[start:end] {
			idx := start + i
			if idx == m.suggestionIdx {
				views = append(views, selectedSuggestionStyle.Render("> "+s))
			} else {
				views = append(views, suggestionStyle.Render("  "+s))
			}
		}
		
		// Wrap suggestions in a container
		content := strings.Join(views, "\n")
		suggestionsView = "\n" + suggestionContainerStyle.Render(content)
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
