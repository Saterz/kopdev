package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

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

	if err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		absolutePath, absolutePathErr := filepath.Abs(path)
		if absolutePathErr != nil {
			return fmt.Errorf("resolve path %q: %w", path, absolutePathErr)
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
	}); err != nil {
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

func runPackage(args []string, config Config) error {
	flags := flag.NewFlagSet("package", flag.ContinueOnError)

	setUsage(
		flags,
		"kopdev package [options] [plugin-directory]",
		"Create a ZIP archive containing the plugin.",
		"\tplugin-directory  Plugin source directory. Defaults to plugin.source_dir in kopdev.toml.",
		"\tkopdev package\n\tkopdev package ./src\n\tkopdev package --output releases",
	)

	output := flags.String("output", config.Package.OutputDir, "output directory of the ZIP file")

	// force := flags.Bool("force", false, "overwrite an existing ZIP file")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	positionalArgs := flags.Args()
	if len(positionalArgs) > 1 {
		return errors.New("usage: kopdev package [options] [plugin-directory]")
	}

	pluginDirectory := config.Plugin.SourceDir

	if len(positionalArgs) == 1 {
		pluginDirectory = positionalArgs[0]
	}

	if pluginDirectory == "" {
		return errors.New("usage: kopdev package [options] [plugin-directory]")
	}

	if err := createPluginZip(config.Plugin.Name, pluginDirectory, *output, config.Package.Exclude); err != nil {
		return err
	}

	fmt.Printf("Packaged plugin to %s\n", filepath.Join(*output, config.Plugin.Name+".zip"))
	return nil
}
