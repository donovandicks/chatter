package fsutil

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestListFiles(t *testing.T) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "fsutil_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

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
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := ListFiles(tmpDir)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	expected := []string{"file1.txt", "sub/file2.go"}

	sort.Strings(got)
	sort.Strings(expected)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %v, got %v", expected, got)
	}
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
		if !reflect.DeepEqual(got, tt.expected) {
			t.Errorf("FilterFiles(%q) = %v; want %v", tt.query, got, tt.expected)
		}
	}
}
