package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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

func validatePluginName(pluginName string) error {
	if pluginName == "" {
		return errors.New(`config: "plugin.name" is required`)
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
	const configPath = "./kopdev.toml"

	config, err := loadConfig(configPath)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	if err := run(os.Args[1:], config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, config Config) error {
	if len(args) == 0 {
		printHelp()
		return errors.New("no command given")
	}

	switch args[0] {
	case "package":
		return runPackage(args[1:], config)

	case "deploy":
		return runDeploy(args[1:], config)

	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}

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

	package      package a KOReader plugin into a ZIP file
	deploy       deploys the plugin to a remote device`)
}
