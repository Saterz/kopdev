package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

var DefaultKOReaderCloneURL = "https://github.com/koreader/koreader.git"
var DefaultKOReaderVersion = "latest"

func cloneKOReader(koreaderCloneURL string, tag string, cloneDir string, debug bool) error {
	parentDir := filepath.Dir(cloneDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}

	// Keep the temporary directory on the same filesystem so rename is atomic.
	stagingDir, err := os.MkdirTemp(parentDir, ".koreader-clone-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stagingDir)

	cloneCommand := exec.Command("git", "clone", "--branch", tag, "--single-branch", "--depth", "1", "--recurse-submodules", "--shallow-submodules", koreaderCloneURL, stagingDir)
	cloneCommand.Stderr = os.Stderr
	if debug {
		cloneCommand.Stdout = os.Stdout
	}

	if err := cloneCommand.Run(); err != nil {
		return fmt.Errorf("clone koreader: %w", err)
	}

	// Publish the cache entry only once cloning is complete.
	if err := os.Rename(stagingDir, cloneDir); err != nil {
		return fmt.Errorf("publish KOReader clone: %w", err)
	}

	return nil
}

func buildKOReader(cloneDir string, debug bool) error {
	buildCommand := exec.Command("./kodev", "build")
	buildCommand.Dir = cloneDir
	if debug {
		buildCommand.Stdout = os.Stdout
		buildCommand.Stderr = os.Stderr
	}
	if err := buildCommand.Run(); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

func injectUserPlugin(pluginName string, srcDir string, runDir string, excluded []string) error {
	userPluginDir := filepath.Join(runDir, "plugins", pluginName+".koplugin")
	if err := os.MkdirAll(userPluginDir, 0o755); err != nil {
		return err
	}

	if err := copyFilteredFiles(srcDir, userPluginDir, excluded); err != nil {
		return fmt.Errorf("copy user plugin: %w", err)
	}

	return nil
}

func runKOReader(runDir string, cloneDir string, kodevArgs []string) error {
	args := []string{"run", "--no-build"}
	args = append(args, kodevArgs...)
	runCommand := exec.Command("./kodev", args...)
	runCommand.Env = append(os.Environ(), "KO_HOME="+runDir)
	runCommand.Dir = cloneDir
	runCommand.Stdout = os.Stdout
	runCommand.Stderr = os.Stderr
	runCommand.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := runCommand.Run(); err != nil {
		return fmt.Errorf("KOReader failed: %w", err)
	}

	return nil
}

func runRunCommand(args []string, config Config) error {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	debug := flags.Bool("debug", false, "Run with debug enabled to troubleshoot issues")

	koreaderCloneURL := flags.String("koreader-clone-url", DefaultKOReaderCloneURL, "KOReader's clone URL")
	koreaderVersion := flags.String("koreader-version", DefaultKOReaderVersion, "KOReader's version")

	if err := flags.Parse(args); err != nil {
		return err
	}

	positionalArgs := flags.Args()

	tempDir, err := os.MkdirTemp("", "kopdev-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	kopdevCacheDir, err := getKopdevCacheDir()
	if err != nil {
		return err
	}

	version := koreaderVersion
	if *version == "latest" {
		version, err = getLatestKOReaderReleaseTagName()
		if err != nil {
			return err
		}
	}
	key := safeFilename(*version)

	cloneDir := filepath.Join(*kopdevCacheDir, "sources", key)

	runDir := filepath.Join(tempDir, "koreader", key)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}

	// Clones KOReader
	info, err := os.Stat(cloneDir)
	switch {
	case os.IsNotExist(err):
		if err := cloneKOReader(
			*koreaderCloneURL, *version, cloneDir, *debug,
		); err != nil {
			return err
		}
	case err != nil:
		return fmt.Errorf("inspect KOReader cache: %w", err)
	case !info.IsDir():
		return fmt.Errorf("KOReader cache path is not a directory: %s", cloneDir)
	}

	// Builds the cloned KOReader instance
	if err := buildKOReader(cloneDir, *debug); err != nil {
		return err
	}

	// Injects the user's plugin which will be tested
	if err := injectUserPlugin(config.Plugin.Name, config.Plugin.SourceDir, runDir, config.Deploy.Exclude); err != nil {
		return err
	}

	if err := runKOReader(runDir, cloneDir, positionalArgs); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	return nil
}
