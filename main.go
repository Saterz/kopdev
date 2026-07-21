package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Plugin  PluginConfig  `toml:"plugin"`
	Package PackageConfig `toml:"package"`
}

type PluginConfig struct {
	Name      string `toml:"name"`
	SourceDir string `toml:"source_dir"`
}

type PackageConfig struct {
	OutputDir string   `toml:"output_dir"`
	Exclude   []string `toml:"exclude"`
}

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

func validatePluginName(pluginName string) error {
	if pluginName == "" {
		return errors.New(`the plugin name is required`)
	}

	if strings.HasSuffix(pluginName, ".koplugin") {
		return errors.New(`the plugin name cannot end with ".koplugin"`)
	}

	return nil
}

func loadConfig(path string) (Config, error) {
	config := Config{
		Package: PackageConfig{
			OutputDir: "dist",
		},
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse %q %w", path, err)
	}

	if err := validatePluginName(config.Plugin.Name); err != nil {
		return Config{}, err
	}

	return config, nil
}

func isSymlink(entry fs.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
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

func createPluginZip(pluginName string, sourceDir string, outputDir string, exclude []string) error {
	sourceDir = filepath.Clean(sourceDir)
	outputDir = filepath.Clean(outputDir)

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	archiveName := pluginName + ".zip"
	outputPath := filepath.Join(outputDir, archiveName)

	absoluteOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}

	zipWriter := zip.NewWriter(outputFile)

	err = filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		absolutePath, absolutePathErr := filepath.Abs(path)
		if absolutePathErr != nil {
			return fmt.Errorf("resolve path %s, %w", path, err)
		}

		if absolutePath == absoluteOutputPath {
			return nil
		}

		relativePath, err := filepath.Rel(sourceDir, path)
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

		zipPath := filepath.Join(pluginName+".koplugin", relativePath)
		zipPath = filepath.ToSlash(zipPath)

		destination, err := zipWriter.Create(zipPath)
		if err != nil {
			return err
		}

		source, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open source file %q: %w", path, err)
		}

		_, copyErr := io.Copy(destination, source)
		closeErr := source.Close()

		if copyErr != nil {
			return fmt.Errorf(
				"copy source file %q into ZIP: %w",
				path,
				copyErr,
			)
		}

		if closeErr != nil {
			return fmt.Errorf(
				"close source file %q: %w",
				path,
				closeErr,
			)
		}

		return nil
	})

	if err != nil {
		zipWriter.Close()
		outputFile.Close()
		os.Remove(outputPath)
		return fmt.Errorf("package plugin: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		outputFile.Close()
		os.Remove(outputPath)
		return fmt.Errorf("finalize ZIP: %w", err)
	}

	if err := outputFile.Close(); err != nil {
		return fmt.Errorf("close ZIP file: %w", err)
	}

	return nil
}

func main() {
	args := os.Args[1:]

	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runWithConfig(args []string, command func([]string, Config) error) error {
	const configPath = "./kopdev.toml"

	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return command(args, config)
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return errors.New("no command given")
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])

	case "package":
		return runWithConfig(args, runPackage)

	case "deploy":
		return runWithConfig(args, runDeploy)

	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}

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
		return fmt.Errorf("unknown template level %q", *level)
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

	return nil
}

func runPackage(args []string, config Config) error {
	flags := flag.NewFlagSet("package", flag.ContinueOnError)

	output := flags.String("output", config.Package.OutputDir, "output directory of the ZIP file")

	// force := flags.Bool("force", false, "overwrite an existing ZIP file")

	if err := flags.Parse(args); err != nil {
		return err
	}

	positionalArgs := flags.Args()

	pluginDirectory := config.Plugin.SourceDir

	if len(positionalArgs) == 1 {
		pluginDirectory = positionalArgs[0]
	}

	if pluginDirectory == "" {
		return errors.New("usage: kopdev package [options] <plugin-directory>")
	}

	if err := createPluginZip(config.Plugin.Name, pluginDirectory, *output, config.Package.Exclude); err != nil {
		return fmt.Errorf("package plugin: %w", err)
	}

	return nil
}

func runDeploy(args []string, config Config) error {
	return nil
}

func printHelp() {
	fmt.Println(`Kopdev is a utility program to help develop KOReader plugins

Usage:

	kopdev <command> [arguments]
	
The commands are:

	init         create a KOReader plugin project from a template
	package      package a KOReader plugin into a ZIP file
	deploy       deploys the plugin to a remote device`)
}
