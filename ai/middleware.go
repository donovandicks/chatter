package ai

import (
	"context"
	"fmt"

	"github.com/donovandicks/chatter/ai/tools"
	"github.com/donovandicks/chatter/internal/auth"
)

// ToolMiddleware is a function that intercepts tool execution.
type ToolMiddleware func(ctx context.Context, tool tools.FunctionTool, args map[string]any, next ToolHandler) (string, error)

// ToolHandler is the final function that executes the tool.
type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

// ChainToolMiddleware chains multiple middleware functions.
func ChainToolMiddleware(mw ...ToolMiddleware) ToolMiddleware {
	return func(ctx context.Context, tool tools.FunctionTool, args map[string]any, next ToolHandler) (string, error) {
		if len(mw) == 0 {
			return next(ctx, args)
		}

		// Wrap the next middleware
		nextMiddleware := func(ctx context.Context, args map[string]any) (string, error) {
			// Create a new chain without the first element
			nextChain := ChainToolMiddleware(mw[1:]...)
			return nextChain(ctx, tool, args, next)
		}

		return mw[0](ctx, tool, args, nextMiddleware)
	}
}

// PermissionRequester defines the interface for requesting user approval.
type PermissionRequester interface {
	RequestApproval(ctx context.Context, action auth.Action) (auth.PermissionLevel, error)
}

// NewPermissionMiddleware creates a middleware that checks permissions.
func NewPermissionMiddleware(pm *auth.PermissionManager, pr PermissionRequester) ToolMiddleware {
	return func(ctx context.Context, tool tools.FunctionTool, args map[string]any, next ToolHandler) (string, error) {
		// Skip check if components are missing
		if pm == nil || pr == nil {
			return next(ctx, args)
		}

		// Get the required action from the tool
		action := tool.RequestPermission(args)

		// Check existing permissions.
		if action.Diff == "" && pm.Check(action) {
			return next(ctx, args)
		}

		// Request approval
		level, err := pr.RequestApproval(ctx, action)
		if err != nil {
			return "", err
		}

		switch level {
		case auth.LevelReject:
			return "", fmt.Errorf("permission denied by user")
		case auth.LevelApproveOnce:
			return next(ctx, args)
		case auth.LevelApproveSession:
			pm.GrantSession(action)
			return next(ctx, args)
		default:
			return "", fmt.Errorf("unknown permission level")
		}
	}
}