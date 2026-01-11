package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/donovandicks/chatter/ai"
)

var (
	// Styles matching the screenshot
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0")). // Light Gray
			Bold(true).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7F9F7F")). // Sage Green/Blue
			Width(20)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")) // Gray

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")) // Pink/Red

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#87D787")) // Green

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6272A4")). // Purple-ish border
			Padding(1, 2).
			MarginTop(1)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Bold(true).
			MarginBottom(1)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E0E0E0")).
				Bold(true)
)

// RenderStats formats the session statistics into a styled string.
func RenderStats(stats ai.SessionStats, showGoodbye bool) string {
	var sb strings.Builder

	// Goodbye Message
	if showGoodbye {
		sb.WriteString("\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("Agent powering down. Goodbye!"))
		sb.WriteString("\n\n")
	}

	// --- Interaction Summary ---
	sb.WriteString(headerStyle.Render("Interaction Summary"))
	sb.WriteString("\n")

	// Session ID
	sb.WriteString(labelStyle.Render("Session ID:"))
	sb.WriteString(valueStyle.Render(stats.SessionID))
	sb.WriteString("\n")

	// Tool Calls
	successCalls := stats.ToolCalls - stats.ToolErrors
	toolCallsStr := fmt.Sprintf("%d ( %s %d %s %d )",
		stats.ToolCalls,
		successStyle.Render("✓"), successCalls,
		errorStyle.Render("✗"), stats.ToolErrors,
	)
	sb.WriteString(labelStyle.Render("Tool Calls:"))
	sb.WriteString(toolCallsStr)
	sb.WriteString("\n")

	// Success Rate
	successRate := 0.0
	if stats.ToolCalls > 0 {
		successRate = float64(successCalls) / float64(stats.ToolCalls) * 100
	}
	rateStr := fmt.Sprintf("%.1f%%", successRate)
	sb.WriteString(labelStyle.Render("Success Rate:"))
	if successRate == 100 {
		sb.WriteString(successStyle.Render(rateStr))
	} else if successRate > 50 {
		sb.WriteString(valueStyle.Render(rateStr))
	} else {
		sb.WriteString(errorStyle.Render(rateStr))
	}
	sb.WriteString("\n\n")

	// --- Performance ---
	sb.WriteString(headerStyle.Render("Performance"))
	sb.WriteString("\n")

	// Wall Time
	wallTime := time.Since(stats.StartTime)
	sb.WriteString(labelStyle.Render("Wall Time:"))
	sb.WriteString(valueStyle.Render(wallTime.Round(time.Millisecond).String()))
	sb.WriteString("\n")

	// Agent Active Time (API + Tools)
	totalActive := stats.ApiDuration + stats.ToolDuration
	sb.WriteString(labelStyle.Render("Agent Active:"))
	sb.WriteString(valueStyle.Render(totalActive.Round(time.Millisecond).String()))
	sb.WriteString("\n")

	// API Time
	apiPercent := 0.0
	if totalActive > 0 {
		apiPercent = float64(stats.ApiDuration) / float64(totalActive) * 100
	}
	sb.WriteString(labelStyle.Render("  » API Time:"))
	sb.WriteString(fmt.Sprintf("%s (%s)",
		valueStyle.Render(stats.ApiDuration.Round(time.Millisecond).String()),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%.1f%%", apiPercent)),
	))
	sb.WriteString("\n")

	// Tool Time
	toolPercent := 0.0
	if totalActive > 0 {
		toolPercent = float64(stats.ToolDuration) / float64(totalActive) * 100
	}
	sb.WriteString(labelStyle.Render("  » Tool Time:"))
	sb.WriteString(fmt.Sprintf("%s (%s)",
		valueStyle.Render(stats.ToolDuration.Round(time.Millisecond).String()),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(fmt.Sprintf("%.1f%%", toolPercent)),
	))
	sb.WriteString("\n\n")

	// --- Model Usage ---
	sb.WriteString(headerStyle.Render("Model Usage"))
	sb.WriteString("\n")

	// Table Header
	// Columns: Model Name (flex), Reqs, Input Tokens, Output Tokens
	col1 := lipgloss.NewStyle().Width(25).Render("Model")
	col2 := lipgloss.NewStyle().Width(6).Align(lipgloss.Right).Render("Reqs")
	col3 := lipgloss.NewStyle().Width(12).Align(lipgloss.Right).Render("Input Tokens")
	col4 := lipgloss.NewStyle().Width(14).Align(lipgloss.Right).Render("Output Tokens")

	sb.WriteString(tableHeaderStyle.Render(col1 + col2 + col3 + col4))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("-", 60)))
	sb.WriteString("\n")

	// Rows
	for model, usage := range stats.ModelUsage {
		c1 := lipgloss.NewStyle().Width(25).Render(model)
		c2 := lipgloss.NewStyle().Width(6).Align(lipgloss.Right).Render(fmt.Sprintf("%d", usage.Requests))
		c3 := lipgloss.NewStyle().Width(12).Align(lipgloss.Right).Render(fmt.Sprintf("%d", usage.InputTokens))
		c4 := lipgloss.NewStyle().Width(14).Align(lipgloss.Right).Render(fmt.Sprintf("%d", usage.OutputTokens))
		sb.WriteString(valueStyle.Render(c1 + c2 + c3 + c4))
		sb.WriteString("\n")
	}

	// If no model usage (e.g. quit immediately), show a placeholder or nothing
	if len(stats.ModelUsage) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true).Render("  No models used."))
		sb.WriteString("\n")
	}

	// Wrap everything in a nice box
	return boxStyle.Render(sb.String())
}
