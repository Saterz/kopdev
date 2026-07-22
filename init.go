package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed all:templates
var templateFiles embed.FS

func runGit(directory string, args ...string) error {
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}
	return nil
}

func runInit(args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)

	level := flags.String("level", "minimal", "template level: full or minimal")

	if err := flags.Parse(args); err != nil {
		return err
	}

	switch *level {
	case "minimal", "full":
	default:
		return fmt.Errorf("unknown template level %q: expected minimal or full", *level)
	}

	positionalArgs := flags.Args()

	if len(positionalArgs) < 1 || len(positionalArgs) > 2 {
		return errors.New("usage: kopdev init [options] <plugin-name> [plugin-directory]")
	}

	pluginName := positionalArgs[0]
	pluginDir := "."

	if len(positionalArgs) == 2 {
		pluginDir = positionalArgs[1]
	}

	if err := validatePluginName(pluginName); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	pluginDir = filepath.Join(pluginDir, pluginName+".koplugin")

	templateRoot, err := fs.Sub(templateFiles, "templates/"+*level)
	if err != nil {
		return fmt.Errorf("load %s template: %w", *level, err)
	}

	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return fmt.Errorf("creating plugin directory: %w", err)
	}

	if err := fs.WalkDir(templateRoot, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		path = filepath.FromSlash(path)

		destination := filepath.Join(pluginDir, path)

		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		contents, err := fs.ReadFile(templateRoot, path)
		if err != nil {
			return fmt.Errorf("read template file %q: %w", path, err)
		}

		replacer := strings.NewReplacer(
			"{{PLUGIN_NAME_UPPER_FIRST}}", upperFirst(pluginName),
			"{{PLUGIN_NAME_LOWER_FIRST}}", lowerFirst(pluginName),
		)

		contents = []byte(replacer.Replace(string(contents)))

		if err := os.WriteFile(destination, contents, 0o644); err != nil {
			return fmt.Errorf("write plugin file %q: %w", path, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("copy %s template: %w", *level, err)
	}

	if err := runGit(pluginDir, "init", "--initial-branch=main"); err != nil {
		return fmt.Errorf("initialize Git repository: %w", err)
	}

	if *level == "full" {
		if err := runGit(pluginDir, "submodule", "add", "https://github.com/koreader/koreader.git", "koreader"); err != nil {
			return fmt.Errorf("add KOReader submodule: %w", err)
		}
	}

	fmt.Printf("Created plugin %q in %s\n", pluginName, pluginDir)
	return nil
}
