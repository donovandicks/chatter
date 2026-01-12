package tools

import (
	"context"
	"os"
	"path/filepath"
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
