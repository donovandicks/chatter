package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/donovandicks/chatter/internal/auth"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PermissionRequestMsg is sent to the UI when the agent needs approval.
type PermissionRequestMsg struct {
	Action        auth.Action
	ResponseChan  chan auth.PermissionLevel
	IsPreApproved bool
}

// UIPermissionRequester implements ai.PermissionRequester using a channel to signal the UI.
type UIPermissionRequester struct {
	RequestChan chan PermissionRequestMsg
	PermManager *auth.PermissionManager
}

// NewUIPermissionRequester creates a new requester.
func NewUIPermissionRequester(pm *auth.PermissionManager) *UIPermissionRequester {
	return &UIPermissionRequester{
		RequestChan: make(chan PermissionRequestMsg),
		PermManager: pm,
	}
}

// RequestApproval sends a request to the UI and waits for the response.
func (r *UIPermissionRequester) RequestApproval(ctx context.Context, action auth.Action) (auth.PermissionLevel, error) {
	respChan := make(chan auth.PermissionLevel)
	select {
	case r.RequestChan <- PermissionRequestMsg{
		Action:        action,
		ResponseChan:  respChan,
		IsPreApproved: r.PermManager != nil && r.PermManager.Check(action),
	}:
	case <-ctx.Done():
		return auth.LevelReject, ctx.Err()
	}

	select {
	case response := <-respChan:
		return response, nil
	case <-ctx.Done():
		return auth.LevelReject, ctx.Err()
	}
}

// HandlePermissionKeyMsg processes key events for the permission modal.
// Returns the permission level chosen and a boolean indicating if a choice was made.
func HandlePermissionKeyMsg(msg tea.KeyMsg) (auth.PermissionLevel, bool) {
	if msg.Type == tea.KeyCtrlC {
		return auth.LevelReject, true
	}

	switch msg.String() {
	case "y", "Y":
		return auth.LevelApproveOnce, true
	case "s", "S":
		return auth.LevelApproveSession, true
	case "n", "N", "esc":
		return auth.LevelReject, true
	}
	return auth.LevelReject, false
}

// StyleDiff applies color coding to a unified diff string.
func StyleDiff(diff string) string {
	if diff == "" {
		return ""
	}
	lines := strings.Split(diff, "\n")
	var styledLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "+") {
			styledLines = append(styledLines, lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(line))
		} else if strings.HasPrefix(line, "-") {
			styledLines = append(styledLines, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(line))
		} else {
			styledLines = append(styledLines, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(line))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, styledLines...)
}

// RenderPermissionInline renders the permission request dialog inline.
func RenderPermissionInline(req PermissionRequestMsg, width int) string {
	action := req.Action
	var prompt string

	switch action.Type {
	case "file":
		prompt = fmt.Sprintf("I need to %s the following file:", action.Operation)
	case "shell":
		prompt = "I need to execute this command:"
	default:
		prompt = fmt.Sprintf("I need to perform '%s' on:", action.Operation)
	}

	helpText := fmt.Sprintf("y: Allow Once  •  s: Allow all %s (Session)  •  n: Deny", action.Category())

	diffView := StyleDiff(action.Diff)

	// Assemble content
	content := lipgloss.JoinVertical(lipgloss.Left,
		PermTitleStyle.Render("Action Review"),
		PermLabelStyle.Render(prompt),
		PermTargetStyle.Render(action.Target),
		diffView,
		PermHelpStyle.Render(helpText),
	)

	// Render the dialog using the full available width
	return PermDialogStyle.Width(width - 2).Render(content)
}
