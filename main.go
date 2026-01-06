package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}

type model struct {
	viewport  viewport.Model
	textInput textinput.Model
	messages  []string
	err       error
}

var (
	senderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	botStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	vp := viewport.New(30, 5)
	vp.SetContent("Welcome to Chatter! Type a message and press Enter.")

	return model{
		textInput: ti,
		viewport:  vp,
		messages:  []string{},
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
	)

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.textInput.Width = msg.Width
		m.viewport.Height = msg.Height - 5 // Leave room for input and borders
		m.viewport.SetContent(strings.Join(m.messages, "\n"))

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textInput.Value() != "" {
				userMsg := senderStyle.Render("You: ") + m.textInput.Value()
				m.messages = append(m.messages, userMsg)
				m.viewport.SetContent(strings.Join(m.messages, "\n"))
				m.textInput.SetValue("")
				m.viewport.GotoBottom()
			}
		}
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textInput.View(),
	) + "\n"
}