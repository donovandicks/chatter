package ai

import (
	"os"
	"testing"
)

func TestLoadAgentPrompt(t *testing.T) {
	// Original agentSystemPrompt is set via embed, so it should not be empty
	// if we are running in the project directory.

	t.Run("no AGENTS.md", func(t *testing.T) {
		// Ensure AGENTS.md doesn't exist in the current directory during test if we want to be sure
		// But we can't easily change CWD safely in parallel tests.
		// However, we can just check if it returns something.

		prompt, err := loadAgentPrompt()
		if err != nil {
			t.Fatalf("loadAgentPrompt failed: %v", err)
		}
		if prompt == "" {
			t.Error("prompt should not be empty")
		}
	})

	t.Run("with AGENTS.md", func(t *testing.T) {
		content := "Special agent instructions"
		err := os.WriteFile("AGENTS.md", []byte(content), 0o644)
		if err != nil {
			t.Fatalf("failed to create AGENTS.md: %v", err)
		}
		defer func() {
			_ = os.Remove("AGENTS.md")
		}()

		prompt, err := loadAgentPrompt()
		if err != nil {
			t.Fatalf("loadAgentPrompt failed: %v", err)
		}

		expectedSuffix := "\n\n" + content
		if len(prompt) < len(expectedSuffix) || prompt[len(prompt)-len(expectedSuffix):] != expectedSuffix {
			t.Errorf("prompt should end with %q", expectedSuffix)
		}
	})
}
