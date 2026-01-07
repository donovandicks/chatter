package auth

import "sync"

// PermissionLevel represents the user's decision for a requested action.
type PermissionLevel int

const (
	LevelAsk PermissionLevel = iota
	LevelReject
	LevelApproveOnce
	LevelApproveSession
)

// Action defines a specific operation that requires permission.
type Action struct {
	Type      string // e.g., "file", "shell"
	Operation string // e.g., "read", "write", "exec"
	Target    string // e.g., "/path/to/file", "ls -la"
}

// PermissionManager manages the state of granted permissions for the current session.
type PermissionManager struct {
	mu                 sync.RWMutex
	sessionPermissions map[Action]bool
}

// NewPermissionManager creates a new instance of PermissionManager.
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		sessionPermissions: make(map[Action]bool),
	}
}

// Check returns true if the action is explicitly approved for the session.
func (pm *PermissionManager) Check(a Action) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.sessionPermissions[a]
}

// GrantSession approves an action for the remainder of the session.
func (pm *PermissionManager) GrantSession(a Action) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.sessionPermissions[a] = true
}

// Revoke removes a previously granted session permission.
func (pm *PermissionManager) Revoke(a Action) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.sessionPermissions, a)
}

// ClearAll revokes all permissions.
func (pm *PermissionManager) ClearAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.sessionPermissions = make(map[Action]bool)
}

// List returns a list of all currently approved session permissions.
func (pm *PermissionManager) List() []Action {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	actions := make([]Action, 0, len(pm.sessionPermissions))
	for a := range pm.sessionPermissions {
		actions = append(actions, a)
	}
	return actions
}
