package config

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// RecursiveFindConfFiles finds all yaml files in dir.
func RecursiveFindConfFiles(dir string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.Walk(dir, func(path string, f fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() {
			// Skip hidden directories inside initial directory
			if strings.HasPrefix(f.Name(), ".") {
				return filepath.SkipDir
			}

			return nil
		}

		// ignore hidden files
		if strings.HasPrefix(f.Name(), ".") {
			return nil
		}

		if strings.HasSuffix(f.Name(), ".yaml") {
			paths = append(paths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return paths, nil
}
