package fsutil

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListFiles(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "fsutil_test")
	assert.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create some files and dirs
	files := []string{
		"file1.txt",
		"sub/file2.go",
		".env",          // Should be ignored
		".git/config",   // Should be ignored
		"vendor/lib.go", // Should be ignored
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		err := os.MkdirAll(filepath.Dir(path), 0o755)
		assert.NoError(t, err)
		err = os.WriteFile(path, []byte(""), 0o644)
		assert.NoError(t, err)
	}

	got, err := ListFiles(tmpDir)
	assert.NoError(t, err, "ListFiles failed")

	expected := []string{"file1.txt", "sub/file2.go"}

	sort.Strings(got)
	sort.Strings(expected)

	assert.Equal(t, expected, got)
}

func TestFilterFiles(t *testing.T) {
	files := []string{"foo.txt", "bar.go", "baz.md", "foobar.rs"}

	tests := []struct {
		query    string
		expected []string
	}{
		{"foo", []string{"foo.txt", "foobar.rs"}},
		{"bar", []string{"bar.go", "foobar.rs"}},
		{"z", []string{"baz.md"}},
		{"", files},
		{"xyz", nil},
	}

	for _, tt := range tests {
		got := FilterFiles(files, tt.query)
		assert.Equal(t, tt.expected, got, "FilterFiles(%q)", tt.query)
	}
}
