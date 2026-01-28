package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteFile_Run(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")

	tool := WriteFile{}
	args := map[string]any{
		"path":    filePath,
		"content": "Hello, World!",
	}

	result, err := tool.Run(context.Background(), args)
	assert.NoError(t, err, "WriteFile.Run() failed")

	assert.Equal(t, "Successfully wrote to "+filePath, result, "Unexpected result")

	// Verify file content
	content, err := os.ReadFile(filePath)
	assert.NoError(t, err, "Failed to read created file")
	assert.Equal(t, "Hello, World!", string(content), "Unexpected content")
}

func TestWriteFile_RequestPermission_Diff(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "diff_test.txt")

	// Create initial file
	initialContent := "line 1\nline 2\nline 3"
	err := os.WriteFile(filePath, []byte(initialContent), 0o644)
	assert.NoError(t, err, "Failed to create file")

	tool := WriteFile{}
	newContent := "line 1\nline 2 modified\nline 3"
	args := map[string]any{
		"path":    filePath,
		"content": newContent,
	}

	action := tool.RequestPermission(args)

	assert.NotEmpty(t, action.Diff, "Expected diff to be generated, but it was empty")

	// Simple check for presence of modification
	assert.Contains(t, action.Diff, "-line 2", "Expected diff to contain deletion of 'line 2'")
	assert.Contains(t, action.Diff, "+line 2 modified", "Expected diff to contain addition of 'line 2 modified'")
}
