package ai

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadAgentPrompt(t *testing.T) {
	// Original agentSystemPrompt is set via embed, so it should not be empty
	// if we are running in the project directory.

	t.Run("no AGENTS.md", func(t *testing.T) {
		// Ensure AGENTS.md doesn't exist in the current directory during test if we want to be sure
		// But we can't easily change CWD safely in parallel tests.
		// However, we can just check if it returns something.

		prompt, err := loadAgentPrompt()
		assert.NoError(t, err, "loadAgentPrompt failed")
		assert.NotEmpty(t, prompt, "prompt should not be empty")
	})

	t.Run("with AGENTS.md", func(t *testing.T) {
		content := "Special agent instructions"
		err := os.WriteFile("AGENTS.md", []byte(content), 0o644)
		assert.NoError(t, err, "failed to create AGENTS.md")
		defer func() {
			_ = os.Remove("AGENTS.md")
		}()

		prompt, err := loadAgentPrompt()
		assert.NoError(t, err, "loadAgentPrompt failed")

		expectedSuffix := "\n\n" + content
		assert.Condition(t, func() bool {
			return len(prompt) >= len(expectedSuffix) && prompt[len(prompt)-len(expectedSuffix):] == expectedSuffix
		}, "prompt should end with %q", expectedSuffix)
	})
}
