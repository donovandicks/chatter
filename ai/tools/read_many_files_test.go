package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

func TestReadManyFiles_Decl(t *testing.T) {
	tool := ReadManyFiles{}
	decl := tool.Decl()

	assert.NotNil(t, decl, "Decl() returned nil")
	assert.Equal(t, "read_many_files", decl.Name, "Expected name 'read_many_files'")
	assert.NotNil(t, decl.Parameters, "Parameters is nil")
	assert.Equal(t, genai.TypeObject, decl.Parameters.Type, "Expected parameter type Object")
}

func TestReadManyFiles_Run(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "test_read_many_files")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create files
	files := map[string]string{
		"file1.txt": "Content 1",
		"file2.go":  "package main\nfunc main() {}",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		assert.NoError(t, err)
	}

	// Create a subdirectory with a file (should be ignored)
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0o755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "ignored.txt"), []byte("Ignored"), 0o644)
	assert.NoError(t, err)

	// Create a hidden file (should be ignored)
	err = os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("Hidden"), 0o644)
	assert.NoError(t, err)

	tool := ReadManyFiles{}
	args := map[string]any{
		"path": tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	assert.NoError(t, err, "Run failed")

	// Verify output
	for name, content := range files {
		expectedPath := filepath.Join(tmpDir, name)
		assert.Contains(t, result, "--- "+expectedPath+" ---", "Result missing file header for %s", name)
		assert.Contains(t, result, content, "Result missing content for %s", name)
	}

	assert.NotContains(t, result, "ignored.txt", "Result should not contain content from subdirectory")
	assert.NotContains(t, result, "Ignored", "Result should not contain content from subdirectory")
	assert.NotContains(t, result, ".hidden", "Result should not contain hidden files")
	assert.NotContains(t, result, "Hidden", "Result should not contain hidden files")
}
