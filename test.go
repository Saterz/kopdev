package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var KOReaderCloneURL = "https://github.com/koreader/koreader.git"
var KOReaderDefaultVersion = "v2026.07"

func koreaderCloned(cloneDir string) bool {
	info, err := os.Stat(filepath.Join(cloneDir, ".git"))
	return err == nil && info.IsDir()
}

func cloneKOReader(tag string, cloneDir string) error {
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		return err
	}
	cloneCommand := exec.Command("git", "clone", "--branch", tag, "--single-branch", "--depth", "1", "--recurse-submodules", "--shallow-submodules", KOReaderCloneURL, cloneDir)
	cloneCommand.Stdout = os.Stdout
	cloneCommand.Stderr = os.Stderr
	if err := cloneCommand.Run(); err != nil {
		return fmt.Errorf("clone: %w", err)
	}
	return nil
}

func buildKOReader() error {
	buildCommand := exec.Command("./kodev", "build")
	buildCommand.Stdout = os.Stdout
	buildCommand.Stderr = os.Stderr
	if err := buildCommand.Run(); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

func runKOReader() error {
	runCommand := exec.Command("./kodev", "run", "--no-build")
	runCommand.Stdout = os.Stdout
	runCommand.Stderr = os.Stderr
	if err := runCommand.Run(); err != nil {
		return fmt.Errorf("run: %w", err)
	}
	return nil
}

func runTest(args []string) error {
	// flags := flag.NewFlagSet("test", flag.ContinueOnError)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	kopdevCacheDir := filepath.Join(cacheDir, "kopdev")

	key := safeFilename(KOReaderDefaultVersion)

	cloneDir := filepath.Join(kopdevCacheDir, "sources", key)

	if !koreaderCloned(cloneDir) {
		if err := cloneKOReader(KOReaderDefaultVersion, cloneDir); err != nil {
			return err
		}
	}
	if err := os.Chdir(cloneDir); err != nil {
		return err
	}
	if err := buildKOReader(); err != nil {
		return err
	}
	if err := runKOReader(); err != nil {
		return err
	}
	return nil
}
