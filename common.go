package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func commandError(err error, output []byte) error {
	message := strings.TrimSpace(string(output))
	if message != "" {
		return fmt.Errorf("%w: %s", err, message)
	}
	return err
}

func setUsage(flags *flag.FlagSet, usage, description, arguments, examples string) {
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n\n%s\n", usage, description)

		if arguments != "" {
			fmt.Fprintf(flags.Output(), "\nArguments:\n%s\n", arguments)
		}

		fmt.Fprintln(flags.Output(), "\nOptions:")
		flags.PrintDefaults()

		if examples != "" {
			fmt.Fprintf(flags.Output(), "\nExamples:\n%s\n", examples)
		}
	}
}

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

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
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

// Copies the plugin source files, which aren't part of the given exclusion array, from a directory to another
func copyFilteredFiles(sourceDir string, outputDir string, exclude []string) error {
	// The comments are here to remind me how this function works because I always forget
	// copyFilteredFiles("/src", "/mnt/plugin-copy", []string{})
	if err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		// func("/src/main.lua", entry, nil)
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(sourceDir, path) // relativePath = "main.lua"
		if err != nil {
			return fmt.Errorf(
				"calculate relative path for %q: %w",
				path,
				err,
			)
		}

		excluded, err := matchesExclude(relativePath, exclude)
		if err != nil {
			return err
		}

		if excluded {
			if entry.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if entry.IsDir() {
			return nil
		}

		if isSymlink(entry) {
			return fmt.Errorf("symbolic links are not supported: %q", path)
		}

		destinationPath := filepath.Join(outputDir, relativePath) // destinationPath = "/mnt/plugin-copy/main.lua"

		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			return err
		}

		destination, err := os.Create(destinationPath)
		if err != nil {
			return err
		}

		source, err := os.Open(path)
		if err != nil {
			destination.Close()
			return fmt.Errorf("read file: %w", err)
		}

		_, err = io.Copy(destination, source)
		if err != nil {
			destination.Close()
			source.Close()
			return err
		}

		if err := destination.Close(); err != nil {
			source.Close()
			return err
		}

		err = source.Close()

		return err
	}); err != nil {
		return err
	}
	return nil
}
