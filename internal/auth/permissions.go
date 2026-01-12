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

// Category returns a string representation of the action's category.
func (a Action) Category() string {
	if a.Type == "" && a.Operation == "" {
		return "unknown"
	}
	return a.Type + ":" + a.Operation
}

// PermissionManager manages the state of granted permissions for the current session.
type PermissionManager struct {
	mu                sync.RWMutex
	sessionActions    map[Action]bool
	sessionCategories map[string]bool
}

// NewPermissionManager creates a new instance of PermissionManager.
func NewPermissionManager() *PermissionManager {
	return &PermissionManager{
		sessionActions:    make(map[Action]bool),
		sessionCategories: make(map[string]bool),
	}
}

// Check returns true if the action or its category is explicitly approved for the session.
func (pm *PermissionManager) Check(a Action) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.sessionActions[a] || pm.sessionCategories[a.Category()]
}

// GrantSession approves an action's category for the remainder of the session.
func (pm *PermissionManager) GrantSession(a Action) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.sessionCategories[a.Category()] = true
}

// Revoke removes a previously granted session permission for an action and its category.
func (pm *PermissionManager) Revoke(a Action) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.sessionActions, a)
	delete(pm.sessionCategories, a.Category())
}

// ClearAll revokes all permissions.
func (pm *PermissionManager) ClearAll() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.sessionActions = make(map[Action]bool)
	pm.sessionCategories = make(map[string]bool)
}

// List returns a list of all currently approved session permissions (actions and categories).
func (pm *PermissionManager) List() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	var list []string
	for a := range pm.sessionActions {
		list = append(list, "Action: "+a.Type+":"+a.Operation+" ("+a.Target+")")
	}
	for c := range pm.sessionCategories {
		list = append(list, "Category: "+c)
	}
	return list
}
