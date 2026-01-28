package tools

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genai"
)

func TestReadFile_Decl(t *testing.T) {
	tool := ReadFile{}
	decl := tool.Decl()

	assert.NotNil(t, decl, "Decl() returned nil")
	assert.Equal(t, "read_file", decl.Name, "Expected name 'read_file'")
	assert.NotNil(t, decl.Parameters, "Parameters is nil")
	assert.Equal(t, genai.TypeObject, decl.Parameters.Type, "Expected parameter type Object")
}

func TestReadFile_Run(t *testing.T) {
	// Create a temp file
	tmpfile, err := os.CreateTemp("", "test_read_file")
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmpfile.Name()) }() // clean up

	content := "Hello, World!"
	_, err = tmpfile.Write([]byte(content))
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	tool := ReadFile{}
	args := map[string]any{
		"path": tmpfile.Name(),
	}

	result, err := tool.Run(context.Background(), args)
	assert.NoError(t, err, "Run failed")
	assert.Equal(t, content, result, "Expected content %q, got %q", content, result)
}
