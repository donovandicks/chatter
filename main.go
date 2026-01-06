package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/donovandicks/chatter/ai"
)

func main() {
	ctx := context.Background()
	agent, err := ai.NewAgent(ctx)
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(initialModel(agent))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}

type model struct {
	viewport  viewport.Model
	textInput textinput.Model
	messages  []string
	agent     *ai.Agent
	spinner   spinner.Model
	isLoading bool
	err       error
}

var (
	senderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	botStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

type agentResponseMsg string
type errMsg error

func initialModel(agent *ai.Agent) model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	vp := viewport.New(30, 5)
	vp.SetContent("Welcome to Chatter! Type a message and press Enter.")

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		textInput: ti,
		viewport:  vp,
		messages:  []string{},
		agent:     agent,
		spinner:   s,
		isLoading: false,
		err:       nil,
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

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	if m.isLoading {
		m.spinner, spCmd = m.spinner.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textInput.Width = msg.Width
		m.viewport.Height = msg.Height - 5
		m.viewport.SetContent(strings.Join(m.messages, "\n"))

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" && !m.isLoading {
				userText := m.textInput.Value()
				userMsg := senderStyle.Render("You: ") + userText
				m.messages = append(m.messages, userMsg)
				m.viewport.SetContent(strings.Join(m.messages, "\n"))
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
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.viewport.GotoBottom()
		return m, tea.Batch(tiCmd, vpCmd)

	case errMsg:
		m.isLoading = false
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m model) View() string {
	if m.isLoading {
		return fmt.Sprintf(
			"%%s\n\n%%s %%s",
			m.viewport.View(),
			m.spinner.View(),
			"Thinking...",
		) + "\n"
	}

	return fmt.Sprintf(
		"%%s\n\n%%s",
		m.viewport.View(),
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
