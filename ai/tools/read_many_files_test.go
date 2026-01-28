package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/genai"
)

func TestReadManyFiles_Decl(t *testing.T) {
	tool := ReadManyFiles{}
	decl := tool.Decl()

	if decl == nil {
		t.Fatal("Decl() returned nil")
	}

	if decl.Name != "read_many_files" {
		t.Errorf("Expected name 'read_many_files', got %q", decl.Name)
	}

	if decl.Parameters == nil {
		t.Fatal("Parameters is nil")
	}

	if decl.Parameters.Type != genai.TypeObject {
		t.Errorf("Expected parameter type Object, got %v", decl.Parameters.Type)
	}
}

func TestReadManyFiles_Run(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "test_read_many_files")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files
	files := map[string]string{
		"file1.txt": "Content 1",
		"file2.go":  "package main\nfunc main() {}",
	}

	for name, content := range files {
		err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Create a subdirectory with a file (should be ignored)
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "ignored.txt"), []byte("Ignored"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a hidden file (should be ignored)
	if err := os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("Hidden"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool := ReadManyFiles{}
	args := map[string]any{
		"path": tmpDir,
	}

	result, err := tool.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	// Verify output
	for name, content := range files {
		expectedPath := filepath.Join(tmpDir, name)
		if !strings.Contains(result, "--- "+expectedPath+" ---") {
			t.Errorf("Result missing file header for %s", name)
		}
		if !strings.Contains(result, content) {
			t.Errorf("Result missing content for %s", name)
		}
	}

	if strings.Contains(result, "ignored.txt") || strings.Contains(result, "Ignored") {
		t.Error("Result should not contain content from subdirectory")
	}

	if strings.Contains(result, ".hidden") || strings.Contains(result, "Hidden") {
		t.Error("Result should not contain hidden files")
	}
}
