package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Temporary ssh url as the repo is currently private. Provide an https clone url when public
var KontrollerCloneURL = "git@github.com:Saterz/Kontroller.git"

// Not in use yet as Kontroller has not yet been released
var KontrollerDefaultVersion = "v1.0.0"

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
	if err := cloneCommand.Run(); err != nil {
		return fmt.Errorf("clone koreader: %w", err)
	}
	return nil
}

func buildKOReader() error {
	buildCommand := exec.Command("./kodev", "build")
	if err := buildCommand.Run(); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

func injectKontroller(tempDir string, runDir string) error {
	kontrollerCloneDir := filepath.Join(tempDir, "kontroller")
	if err := os.MkdirAll(kontrollerCloneDir, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(kontrollerCloneDir)

	cloneCommand := exec.Command("git", "clone", "--depth", "1", KontrollerCloneURL, kontrollerCloneDir)
	if err := cloneCommand.Run(); err != nil {
		return fmt.Errorf("clone kontroller: %w", err)
	}

	kontrollerPluginsDir := filepath.Join(kontrollerCloneDir, "plugins")
	kontrollerPatchesDir := filepath.Join(kontrollerCloneDir, "patches")

	runPluginsDir := filepath.Join(runDir, "plugins")
	runPatchesDir := filepath.Join(runDir, "patches")

	if err := os.Mkdir(runPluginsDir, 0o755); err != nil {
		return err
	}
	if err := os.Mkdir(runPatchesDir, 0o755); err != nil {
		return err
	}

	copyFilteredFiles(kontrollerPluginsDir, runPluginsDir, nil)
	copyFilteredFiles(kontrollerPatchesDir, runPatchesDir, nil)

	return nil
}

func runKOReader(runDir string, logsPath string) error {
	logsFile, err := os.Create(logsPath)
	if err != nil {
		return err
	}
	defer logsFile.Close()

	runCommand := exec.Command("./kodev", "run", "--no-build")
	runCommand.Env = append(os.Environ(), "KO_HOME="+runDir)
	runCommand.Stdout = logsFile
	runCommand.Stderr = logsFile
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
	tempDir, err := os.MkdirTemp("", "kopdev-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	kopdevCacheDir := filepath.Join(cacheDir, "kopdev")

	key := safeFilename(KOReaderDefaultVersion)

	cloneDir := filepath.Join(kopdevCacheDir, "sources", key)
	logsPath := filepath.Join(kopdevCacheDir, "logs", key+".log")
	if err := os.MkdirAll(filepath.Dir(logsPath), 0o755); err != nil {
		return err
	}

	runDir := filepath.Join(tempDir, "koreader", key)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil
	}

	// Clones KOReader is not done already
	if !koreaderCloned(cloneDir) {
		if err := cloneKOReader(KOReaderDefaultVersion, cloneDir); err != nil {
			return err
		}
	}

	// Moves the process to the new cloned directory (needs thinking)
	if err := os.Chdir(cloneDir); err != nil {
		return err
	}

	// Builds the cloned KOReader instance
	if err := buildKOReader(); err != nil {
		return err
	}

	// Injects the Kontroller plugin and patches
	if err := injectKontroller(tempDir, runDir); err != nil {
		return err
	}

	// Runs KOReader
	if err := runKOReader(runDir, logsPath); err != nil {
		return err
	}
	return nil
}
