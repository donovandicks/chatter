package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Chat message styles
	senderNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("5")).
			Bold(true).
			MarginRight(1)

	botNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Bold(true).
			MarginRight(1)

	messageBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1).
			MarginBottom(1)

	senderBoxStyle = messageBoxStyle.
			BorderForeground(lipgloss.Color("63")) // Purple-ish

	botBoxStyle = messageBoxStyle.
			BorderForeground(lipgloss.Color("39")) // Blue-ish

	// Autocomplete styles
	suggestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedSuggestionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true).
				Background(lipgloss.Color("237")).
				Padding(0, 1)

	suggestionContainerStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("240")).
					Background(lipgloss.Color("235")).
					MarginBottom(1)
)

// renderMessage wraps the content in a styled box.
func renderMessage(sender, content string, isUser bool, width int) string {
	nameStyle := botNameStyle
	boxStyle := botBoxStyle
	if isUser {
		nameStyle = senderNameStyle
		boxStyle = senderBoxStyle
	}

	// Calculate inner width for the content
	// Border (2) + Padding (2) = 4
	innerWidth := max(width-4, 10)

	// Create the header (sender name)
	header := nameStyle.Render(sender)

	// Render the content with wrapping
	contentStyle := lipgloss.NewStyle().Width(innerWidth)
	renderedContent := contentStyle.Render(content)

	// Render the box with the specified width
	return boxStyle.
		Width(width - 2). // -2 for right margin/gutter safety
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				header,
				renderedContent,
			),
		)
}
