package ai

import (
	"testing"

	"github.com/donovandicks/chatter/ai/tools"
)

func TestAgentRegistry(t *testing.T) {
	r := NewAgentRegistry()

	// 1. Initial state
	if len(r.List()) != 0 {
		t.Errorf("Expected empty registry, got %d items", len(r.List()))
	}

	// 2. Register agent
	cfg := AgentConfig{
		Name:         "test_agent",
		Model:        Gemini3Flash,
		SystemPrompt: "test prompt",
		Tools:        []tools.FunctionTool{},
	}
	r.Register(cfg)

	// 3. List
	list := r.List()
	if len(list) != 1 {
		t.Errorf("Expected 1 item, got %d", len(list))
	}
	if list[0] != "test_agent" {
		t.Errorf("Expected 'test_agent', got %s", list[0])
	}

	// 4. Get existing
	retrieved, ok := r.Get("test_agent")
	if !ok {
		t.Error("Expected to find agent")
	}
	if retrieved.Name != "test_agent" {
		t.Errorf("Expected name 'test_agent', got %s", retrieved.Name)
	}

	// 5. Get non-existing
	_, ok = r.Get("non_existent")
	if ok {
		t.Error("Expected not to find agent")
	}
}
