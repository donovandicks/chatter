package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if err != nil {
		t.Fatalf("WriteFile.Run() failed: %v", err)
	}

	if result != "Successfully wrote to "+filePath {
		t.Errorf("Unexpected result: %s", result)
	}

	// Verify file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestWriteFile_RequestPermission_Diff(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "diff_test.txt")

	// Create initial file
	initialContent := "line 1\nline 2\nline 3"
	if err := os.WriteFile(filePath, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	tool := WriteFile{}
	newContent := "line 1\nline 2 modified\nline 3"
	args := map[string]any{
		"path":    filePath,
		"content": newContent,
	}

	action := tool.RequestPermission(args)

	if action.Diff == "" {
		t.Error("Expected diff to be generated, but it was empty")
	}

	// Simple check for presence of modification
	if !strings.Contains(action.Diff, "-line 2") {
		t.Errorf("Expected diff to contain deletion of 'line 2', got: %s", action.Diff)
	}
	if !strings.Contains(action.Diff, "+line 2 modified") {
		t.Errorf("Expected diff to contain addition of 'line 2 modified', got: %s", action.Diff)
	}
}
