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

	senderBoxStyle = messageBoxStyle.Copy().
			BorderForeground(lipgloss.Color("63")) // Purple-ish

	botBoxStyle = messageBoxStyle.Copy().
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
	innerWidth := width - 4
	if innerWidth < 10 {
		innerWidth = 10 // Minimum width safety
	}

	// Create the header (sender name)
	header := nameStyle.Render(sender)

	// Render the content with wrapping
	contentStyle := lipgloss.NewStyle().Width(innerWidth)
	renderedContent := contentStyle.Render(content)

	// Render the box with the specified width
	return boxStyle.
		Width(width - 2). // Account for outer margins/imperfections? Lipgloss Border adds to width if not specified?
		// Actually, if we set Width on the style, it includes content + padding, but border is added outside if standard box model isn't used?
		// Lipgloss default is: Width applies to the content area unless `Border` is used, then it depends.
		// Let's set Width on the style which usually enforces the *content* width if not careful,
		// but `Width` on a style with Border usually sets the total width if using standard sizing or just content?
		// Safest is to set width on the content and let the box wrap it.
		// But to force the box to expand to the viewport:
		Width(width - 2). // -2 for right margin/gutter safety
		Render(
			lipgloss.JoinVertical(lipgloss.Left,
				header,
				renderedContent,
			),
		)
}
