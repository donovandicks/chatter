package ui

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// listFiles returns a list of relative file paths in the given root directory.
// It ignores .git directories.
func listFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Ignore .git, node_modules, vendor directories
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" || d.Name() == "vendor" {
				return filepath.SkipDir
			}
		}

		if !d.IsDir() {
			// Ignore .env files
			if d.Name() == ".env" {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// filterFiles returns a filtered list of files that match the query.
func filterFiles(files []string, query string) []string {
	if query == "" {
		return files
	}

	var filtered []string
	for _, f := range files {
		if strings.Contains(strings.ToLower(f), strings.ToLower(query)) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
