package ui

import (
	"context"

	"github.com/donovandicks/chatter/internal/auth"
)

// PermissionRequestMsg is sent to the UI when the agent needs approval.
type PermissionRequestMsg struct {
	Action       auth.Action
	ResponseChan chan auth.PermissionLevel
}

// UIPermissionRequester implements ai.PermissionRequester using a channel to signal the UI.
type UIPermissionRequester struct {
	RequestChan chan PermissionRequestMsg
}

// NewUIPermissionRequester creates a new requester.
func NewUIPermissionRequester() *UIPermissionRequester {
	return &UIPermissionRequester{
		RequestChan: make(chan PermissionRequestMsg),
	}
}

// RequestApproval sends a request to the UI and waits for the response.
func (r *UIPermissionRequester) RequestApproval(ctx context.Context, action auth.Action) (auth.PermissionLevel, error) {
	respChan := make(chan auth.PermissionLevel)
	select {
	case r.RequestChan <- PermissionRequestMsg{
		Action:       action,
		ResponseChan: respChan,
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
