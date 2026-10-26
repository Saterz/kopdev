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
	"regexp"
	"strings"
)

//go:embed all:templates
var templateFiles embed.FS

var projectNamePattern = regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

func runGit(directory string, args ...string) error {
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return commandError(err, output)
	}
	return nil
}

func runInit(args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)

	setUsage(
		flags,
		"kopdev init [options] <plugin-name> [plugin-directory]",
		"Create a new KOReader plugin project.",
		"\tplugin-name       Name of the plugin without the .koplugin suffix.\n\tplugin-directory  Parent directory for the project. Defaults to \".\".",
		"\tkopdev init my-awesome-plugin\n\tkopdev init --project plugin my_awesome_plugin ./plugins",
	)

	project := flags.String("project", "plugin", "template project: plugin or patch")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	switch *project {
	case "plugin", "patch":
	default:
		return fmt.Errorf("unknown template level %q: expected plugin or patch", *project)
	}

	positionalArgs := flags.Args()

	if len(positionalArgs) < 1 || len(positionalArgs) > 2 {
		return errors.New("usage: kopdev init [options] <plugin-name> [plugin-directory]")
	}

	projectName := positionalArgs[0]
	projectDir := "."

	if !projectNamePattern.MatchString(projectName) {
		return errors.New("project name must start with a letter or underscore and contain only letters, digits, and underscores")
	}

	if len(positionalArgs) == 2 {
		projectDir = positionalArgs[1]
	}

	if err := validatePluginName(projectName); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	switch *project {
	case "plugin":
		projectDir = filepath.Join(projectDir, projectName+".koplugin")
	case "patch":
		projectDir = filepath.Join(projectDir, projectName+".patches")
	}

	templateRoot, err := fs.Sub(templateFiles, "templates/"+*project)
	if err != nil {
		return fmt.Errorf("load %s template: %w", *project, err)
	}

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	if err := fs.WalkDir(templateRoot, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		path = filepath.FromSlash(path)

		destination := filepath.Join(projectDir, path)

		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}

		contents, err := fs.ReadFile(templateRoot, path)
		if err != nil {
			return fmt.Errorf("read template file %q: %w", path, err)
		}

		replacer := strings.NewReplacer(
			"TEMP_PLUGIN_NAME_UPPER", upperFirst(projectName),
			"TEMP_PLUGIN_NAME_LOWER", lowerFirst(projectName),
		)

		contents = []byte(replacer.Replace(string(contents)))

		if err := os.WriteFile(destination, contents, 0o644); err != nil {
			return fmt.Errorf("write plugin file %q: %w", path, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("copy %s template: %w", *project, err)
	}

	if err := runGit(projectDir, "init", "--initial-branch=main"); err != nil {
		return fmt.Errorf("initialize Git repository: %w", err)
	}

	fmt.Printf("Created plugin %q in %s\n", projectName, projectDir)
	return nil
}
