package main

import (
	"fmt"
	"path/filepath"
	"unicode"
)

func upperFirst(value string) string {
	characters := []rune(value)
	if len(characters) > 0 {
		characters[0] = unicode.ToUpper(characters[0])
	}
	return string(characters)
}

func lowerFirst(value string) string {
	characters := []rune(value)
	if len(characters) > 0 {
		characters[0] = unicode.ToLower(characters[0])
	}
	return string(characters)
}

func matchesExclude(path string, exclude []string) (bool, error) {
	path = filepath.Clean(path)
	fileName := filepath.Base(path)

	for _, pattern := range exclude {
		pattern = filepath.Clean(pattern)

		pathMatched, err := filepath.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
		}

		if pathMatched {
			return true, nil
		}

		nameMatched, err := filepath.Match(pattern, fileName)
		if err != nil {
			return false, fmt.Errorf("invalid exclusion pattern %q: %w", pattern, err)
		}

		if nameMatched {
			return true, nil
		}
	}

	return false, nil
}
