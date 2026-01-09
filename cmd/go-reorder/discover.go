package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

func discoverFiles(paths []string, excludePatterns []string) ([]string, error) {
	var files []string

	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}

		if !info.IsDir() {
			// Single file
			if strings.HasSuffix(p, ".go") {
				if !isExcluded(p, excludePatterns) {
					files = append(files, p)
				}
			}
			continue
		}

		// Directory: walk recursively
		baseDir := p
		err = filepath.WalkDir(p, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".go") {
				// Get relative path for pattern matching
				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					relPath = path
				}
				if !isExcluded(relPath, excludePatterns) {
					files = append(files, path)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

// isExcluded checks if a path matches any of the exclude patterns.
func isExcluded(path string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also check the base name for patterns like "*_test.go"
		if matched, err := doublestar.Match(pattern, filepath.Base(path)); err == nil && matched {
			return true
		}
	}
	return false
}
