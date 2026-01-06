package tools

import (
	"os"
	"testing"

	"google.golang.org/genai"
)

func TestReadFile_Decl(t *testing.T) {
	tool := ReadFile{}
	decl := tool.Decl()

	if decl == nil {
		t.Fatal("Decl() returned nil")
	}

	if decl.Name != "read_file" {
		t.Errorf("Expected name 'read_file', got %q", decl.Name)
	}

	if decl.Parameters == nil {
		t.Fatal("Parameters is nil")
	}
	
	if decl.Parameters.Type != genai.TypeObject {
		t.Errorf("Expected parameter type Object, got %v", decl.Parameters.Type)
	}
}

func TestReadFile_Run(t *testing.T) {
	// Create a temp file
	tmpfile, err := os.CreateTemp("", "test_read_file")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	content := "Hello, World!"
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	tool := ReadFile{}
	args := map[string]any{
		"path": tmpfile.Name(),
	}

	result, err := tool.Run(args)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result != content {
		t.Errorf("Expected content %q, got %q", content, result)
	}
}
