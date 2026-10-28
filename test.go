package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Saterz/kopdev/internal/kontroller"
	"github.com/ybbus/jsonrpc/v3"
)

var DefaultKOReaderCloneURL = "https://github.com/koreader/koreader.git"
var DefaultKOReaderVersion = "v2026.07"

const koreaderShutdownTimeout = 5 * time.Second

func findEmulatorDir(cloneDir string) (string, error) {
	pattern := filepath.Join(cloneDir, "koreader-emulator-*-debug")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("find emulator directory: %w", err)
	}

	if len(matches) != 1 {
		return "", fmt.Errorf("expected one emulator directory matching %q, found %d", pattern, len(matches))
	}

	return filepath.Join(matches[0], "koreader"), nil
}

func koreaderCloned(cloneDir string) bool {
	info, err := os.Stat(filepath.Join(cloneDir, ".git"))
	return err == nil && info.IsDir()
}

func cloneKOReader(koreaderCloneURL string, tag string, cloneDir string, debug bool) error {
	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		return err
	}
	cloneCommand := exec.Command("git", "clone", "--branch", tag, "--single-branch", "--depth", "1", "--recurse-submodules", "--shallow-submodules", koreaderCloneURL, cloneDir)
	if debug {
		cloneCommand.Stdout = os.Stdout
	}
	if err := cloneCommand.Run(); err != nil {
		return fmt.Errorf("clone koreader: %w", err)
	}
	return nil
}

func buildKOReader(cloneDir string, debug bool) error {
	buildCommand := exec.Command("./kodev", "build")
	buildCommand.Dir = cloneDir
	if debug {
		buildCommand.Stdout = os.Stdout
	}
	if err := buildCommand.Run(); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

func injectKontroller(tempDir string, runDir string) (string, error) {
	kontrollerDestinationDir := filepath.Join(tempDir, "kontroller")
	if err := os.MkdirAll(kontrollerDestinationDir, 0o755); err != nil {
		return "", err
	}

	source, err := fs.Sub(kontroller.KontrollerFiles, "src")
	if err != nil {
		return "", fmt.Errorf("load Kontroller: %w", err)
	}

	if err := fs.WalkDir(source, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		target := filepath.Join(kontrollerDestinationDir, filepath.FromSlash(path))

		if entry.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create directory %q: %s", target, err)
			}
			return nil
		}

		data, err := fs.ReadFile(source, path)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		if err := os.WriteFile(target, data, 0o644); err != nil {
			return fmt.Errorf("write file %q: %w", target, err)
		}

		return nil
	}); err != nil {
		return "", fmt.Errorf("copy Kontroller: %w", err)
	}

	kontrollerPluginsDir := filepath.Join(kontrollerDestinationDir, "plugins")
	kontrollerPatchesDir := filepath.Join(kontrollerDestinationDir, "patches")

	runPluginsDir := filepath.Join(runDir, "plugins")
	runPatchesDir := filepath.Join(runDir, "patches")

	// The patches folder is not created automatically by KOReader
	if err := os.MkdirAll(runPatchesDir, 0o755); err != nil {
		return "", err
	}

	if err := copyFilteredFiles(kontrollerPluginsDir, runPluginsDir, nil); err != nil {
		return "", fmt.Errorf("copy Kontroller plugin: %w", err)
	}
	if err := copyFilteredFiles(kontrollerPatchesDir, runPatchesDir, nil); err != nil {
		return "", fmt.Errorf("copy Kontroller patches: %w", err)
	}

	return kontrollerDestinationDir, nil
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

func injectUserFixtures(specDir string, runDir string) error {
	fixtureDir := filepath.Join(specDir, "fixtures")
	if info, err := os.Stat(fixtureDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("test fixtures path is not a directory: %q", fixtureDir)
		}
		if err := copyFiles(fixtureDir, runDir); err != nil {
			return fmt.Errorf("copy test fixtures: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect test fixtures: %w", err)
	}

	return nil
}

func startKOReader(runDir string, cloneDir string, logsPath string) (*exec.Cmd, *os.File, error) {
	logsFile, err := os.Create(logsPath)
	if err != nil {
		return nil, nil, err
	}

	runCommand := exec.Command("./kodev", "run", "--no-build")
	runCommand.Env = append(os.Environ(), "KO_HOME="+runDir)
	runCommand.Dir = cloneDir
	runCommand.Stdout = logsFile
	runCommand.Stderr = logsFile
	runCommand.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := runCommand.Start(); err != nil {
		return nil, nil, fmt.Errorf("start KOReader: %w", err)
	}

	return runCommand, logsFile, nil
}

// Wait until Kontroller is able to accept and respond to requests
func waitUntilKontrollerReady(timeout time.Duration) error {
	rpcClient := jsonrpc.NewClient("http://127.0.0.1:8060/jsonrpc")

	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		response, err := rpcClient.Call(context.Background(), "ping")
		if err != nil {
			lastErr = err
			time.Sleep(time.Second)
			continue
		}

		if response.Error != nil {
			return fmt.Errorf("jsonrpc error: %d: %s", response.Error.Code, response.Error.Message)
		}

		result, err := response.GetString()
		if err != nil {
			return err
		}

		if result == "pong" {
			return nil
		}

		time.Sleep(time.Second)
	}

	if lastErr != nil {
		return fmt.Errorf("Kontroller did not become ready: %w", lastErr)
	}
	return fmt.Errorf("Kontroller did not become ready")
}

func exitKOReader(command *exec.Cmd) error {
	rpcClient := jsonrpc.NewClient("http://127.0.0.1:8060/jsonrpc")

	ctx, cancel := context.WithTimeout(context.Background(), koreaderShutdownTimeout)
	defer cancel()

	response, err := rpcClient.Call(ctx, "system.exit")
	if err != nil {
		return err
	}

	if response.Error != nil {
		return fmt.Errorf("jsonrpc error: %d: %s", response.Error.Code, response.Error.Message)
	}

	return waitForKOReaderExit(command)
}

func stopKOReader(command *exec.Cmd) error {
	if command == nil || command.Process == nil {
		return errors.New("KOReader process was not started")
	}

	processGroupID := -command.Process.Pid

	if err := syscall.Kill(processGroupID, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("stop KOReader process group: %w", err)
	}

	return waitForKOReaderExit(command)
}

func waitForKOReaderExit(command *exec.Cmd) error {
	waitResult := make(chan error, 1)

	go func() {
		waitResult <- command.Wait()
	}()

	var waitError error
	select {
	case waitError = <-waitResult:
	case <-time.After(koreaderShutdownTimeout):
		processGroupID := -command.Process.Pid
		if err := syscall.Kill(processGroupID, syscall.SIGKILL); err != nil &&
			!errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("kill stuck KOReader process group: %w", err)
		}
		waitError = <-waitResult
	}

	var exitError *exec.ExitError
	if waitError != nil && !errors.As(waitError, &exitError) {
		return fmt.Errorf("wait for KOReader: %w", waitError)
	}

	return nil
}

func runTest(args []string, config Config) error {
	flags := flag.NewFlagSet("test", flag.ContinueOnError)

	debug := flags.Bool("debug", false, "Run with debug enabled to troubleshoot issues")

	koreaderCloneURL := flags.String("koreader-clone-url", DefaultKOReaderCloneURL, "KOReader's clone URL")
	koreaderVersion := flags.String("koreader-version", DefaultKOReaderVersion, "KOReader's version")

	if err := flags.Parse(args); err != nil {
		return err
	}

	positionalArgs := flags.Args()

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp("", "kopdev-*")
	if err != nil {
		return err
	}

	kopdevCacheDir := filepath.Join(cacheDir, "kopdev")

	key := safeFilename(*koreaderVersion)

	cloneDir := filepath.Join(kopdevCacheDir, "sources", key)
	logsPath := filepath.Join(kopdevCacheDir, "logs", key+".log")
	if err := os.MkdirAll(filepath.Dir(logsPath), 0o755); err != nil {
		return err
	}

	runDir := filepath.Join(tempDir, "koreader", key)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get project directory: %w", err)
	}
	specDir := filepath.Join(projectDir, "specs")

	// Clones KOReader if not done already
	if !koreaderCloned(cloneDir) {
		if err := cloneKOReader(*koreaderCloneURL, *koreaderVersion, cloneDir, *debug); err != nil {
			return err
		}
	}

	// Builds the cloned KOReader instance
	if err := buildKOReader(cloneDir, *debug); err != nil {
		return err
	}

	// Injects the Kontroller plugin and patches
	kontrollerDir, err := injectKontroller(tempDir, runDir)
	if err != nil {
		return err
	}

	// Injects the user's plugin which will be tested
	if err := injectUserPlugin(config.Plugin.Name, config.Plugin.SourceDir, runDir, config.Deploy.Exclude); err != nil {
		return err
	}

	// Injects optional user fixtures
	if err := injectUserFixtures(specDir, runDir); err != nil {
		return err
	}

	koreaderStartCommand, logsFile, err := startKOReader(runDir, cloneDir, logsPath)
	if err != nil {
		return err
	}
	defer func() {
		if koreaderStartCommand != nil {
			_ = stopKOReader(koreaderStartCommand)
		}
	}()

	emulatorDir, err := findEmulatorDir(cloneDir)
	if err != nil {
		return err
	}

	if err := waitUntilKontrollerReady(6 * time.Second); err != nil {
		return err
	}

	testError := runUserTests(emulatorDir, kontrollerDir, specDir, runDir, config, positionalArgs)

	exitErr := exitKOReader(koreaderStartCommand)
	koreaderStartCommand = nil
	// cleanupErr := os.RemoveAll(tempDir)
	logsErr := logsFile.Close()

	return errors.Join(testError, exitErr, logsErr)
}

func runUserTests(emulatorDir string, kontrollerDir string, specDir string, runDir string, config Config, bustedArgs []string) error {
	luaPath := strings.Join([]string{
		"?.lua",
		"common/?.lua",
		"frontend/?.lua",
		"spec/rocks/share/lua/5.1/?.lua",
		"spec/rocks/share/lua/5.1/?/init.lua",
		filepath.Join(filepath.Dir(kontrollerDir), "?.lua"),
		filepath.Join(filepath.Dir(kontrollerDir), "?", "init.lua"),
		filepath.Join(runDir, "plugins", config.Plugin.Name+".koplugin", "?.lua"),
		filepath.Join(runDir, "plugins", config.Plugin.Name+".koplugin", "?", "init.lua"),
	}, ";")

	luaCPath := strings.Join([]string{
		"?.so",
		"common/?.so",
		"spec/rocks/lib/lua/5.1/?.so",
	}, ";")

	args := []string{
		"-e",
		`require "busted.runner" { standalone = false }`,
		"/dev/null",
		"--output=gtest",
		"--loaders=lua",
		"--sort-files",
	}
	args = append(args, bustedArgs...)
	args = append(args, "--", specDir)

	command := exec.Command("./luajit", args...)

	command.Dir = emulatorDir
	command.Env = append(
		os.Environ(),
		"LUA_PATH="+luaPath,
		"LUA_CPATH="+luaCPath,
		"RUN_DIR="+runDir,
		"PLUGIN_NAME="+config.Plugin.Name,
	)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}

	return nil
}
