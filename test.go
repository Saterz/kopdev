package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ybbus/jsonrpc/v3"
)

// Temporary ssh url as the repo is currently private. Provide an https clone url when public
var KontrollerCloneURL = "git@github.com:Saterz/Kontroller.git"

// Not in use yet as Kontroller has not yet been released
var KontrollerDefaultVersion = "v1.0.0"

var KontrollerEndpointURL = "http://127.0.0.1:8060/jsonrpc"

var KOReaderCloneURL = "https://github.com/koreader/koreader.git"
var KOReaderDefaultVersion = "v2026.07"

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

func buildKOReader(cloneDir string) error {
	buildCommand := exec.Command("./kodev", "build")
	buildCommand.Dir = cloneDir
	if err := buildCommand.Run(); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	return nil
}

func injectKontroller(tempDir string, runDir string) (string, error) {
	kontrollerCloneDir := filepath.Join(tempDir, "kontroller")
	if err := os.MkdirAll(kontrollerCloneDir, 0o755); err != nil {
		return "", err
	}

	cloneCommand := exec.Command("git", "clone", "--depth", "1", KontrollerCloneURL, kontrollerCloneDir)
	if err := cloneCommand.Run(); err != nil {
		return "", fmt.Errorf("clone kontroller: %w", err)
	}

	kontrollerPluginsDir := filepath.Join(kontrollerCloneDir, "plugins")
	kontrollerPatchesDir := filepath.Join(kontrollerCloneDir, "patches")

	runPluginsDir := filepath.Join(runDir, "plugins")
	runPatchesDir := filepath.Join(runDir, "patches")

	if err := os.Mkdir(runPluginsDir, 0o755); err != nil {
		return "", err
	}
	if err := os.Mkdir(runPatchesDir, 0o755); err != nil {
		return "", err
	}

	if err := copyFilteredFiles(kontrollerPluginsDir, runPluginsDir, nil); err != nil {
		return "", fmt.Errorf("copy Kontroller plugin: %w", err)
	}
	if err := copyFilteredFiles(kontrollerPatchesDir, runPatchesDir, nil); err != nil {
		return "", fmt.Errorf("copy Kontroller patches: %w", err)
	}

	return kontrollerCloneDir, nil
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
	rpcClient := jsonrpc.NewClient(KontrollerEndpointURL)

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
	rpcClient := jsonrpc.NewClient(KontrollerEndpointURL)

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
	// flags := flag.NewFlagSet("test", flag.ContinueOnError)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp("", "kopdev-*")
	if err != nil {
		return err
	}

	kopdevCacheDir := filepath.Join(cacheDir, "kopdev")

	key := safeFilename(KOReaderDefaultVersion)

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
		if err := cloneKOReader(KOReaderDefaultVersion, cloneDir); err != nil {
			return err
		}
	}

	// Builds the cloned KOReader instance
	if err := buildKOReader(cloneDir); err != nil {
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

	testError := runUserTests(emulatorDir, kontrollerDir, specDir, runDir, config)

	exitErr := exitKOReader(koreaderStartCommand)
	koreaderStartCommand = nil
	cleanupErr := os.RemoveAll(tempDir)
	logsErr := logsFile.Close()

	return errors.Join(testError, exitErr, cleanupErr, logsErr)
}

func runUserTests(emulatorDir string, kontrollerDir string, specDir string, runDir string, config Config) error {
	luaPath := strings.Join([]string{
		"?.lua",
		"common/?.lua",
		"frontend/?.lua",
		"spec/rocks/share/lua/5.1/?.lua",
		"spec/rocks/share/lua/5.1/?/init.lua",
		filepath.Join(kontrollerDir, "src", "?.lua"),
		filepath.Join(kontrollerDir, "src", "?", "init.lua"),
		filepath.Join(runDir, "plugins", config.Plugin.Name+".koplugin", "?.lua"),
		filepath.Join(runDir, "plugins", config.Plugin.Name+".koplugin", "?", "init.lua"),
	}, ";")

	luaCPath := strings.Join([]string{
		"?.so",
		"common/?.so",
		"spec/rocks/lib/lua/5.1/?.so",
	}, ";")

	command := exec.Command(
		"./luajit",
		"-e",
		`require "busted.runner" { standalone = false }`,
		"/dev/null",
		"--output=gtest",
		"--loaders=lua",
		"--sort-files",
		"--",
		specDir,
	)

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
