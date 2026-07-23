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

	"github.com/Saterz/kopdev/internal/minify"
)

type PackageOptions struct {
	PluginName string
	SourceDir  string
	OutputDir  string
	Exclude    []string
	Minify     bool
}

func createPluginZip(options PackageOptions) (err error) {
	options.SourceDir = filepath.Clean(options.SourceDir)
	options.OutputDir = filepath.Clean(options.OutputDir)

	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	archiveName := options.PluginName + ".zip"
	outputPath := filepath.Join(options.OutputDir, archiveName)

	absoluteOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}

	zipWriter := zip.NewWriter(outputFile)
	complete := false
	defer func() {
		if complete {
			return
		}
		zipWriter.Close()
		outputFile.Close()
		if removeErr := os.Remove(outputPath); removeErr != nil && !os.IsNotExist(removeErr) {
			if err == nil {
				err = fmt.Errorf("remove incomplete ZIP %q: %w", outputPath, removeErr)
			} else {
				err = fmt.Errorf("%w; remove incomplete ZIP %q: %v", err, outputPath, removeErr)
			}
		}
	}()

	var luaMinifier *minify.LuaMinifier
	if options.Minify {
		luaMinifier, err = minify.NewLuaMinifier()
		if err != nil {
			return err
		}
		defer luaMinifier.Close()
	}

	if err := filepath.WalkDir(options.SourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
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

		relativePath, err := filepath.Rel(options.SourceDir, path)
		if err != nil {
			return fmt.Errorf(
				"calculate relative path for %q: %w",
				path,
				err,
			)
		}

		excluded, err := matchesExclude(relativePath, options.Exclude)
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

		zipPath := filepath.Join(options.PluginName+".koplugin", relativePath)
		zipPath = filepath.ToSlash(zipPath)

		destination, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("create packaged file %q: %w", zipPath, err)
		}

		if options.Minify && filepath.Ext(path) == ".lua" {
			source, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read source file %q: %w", path, err)
			}

			minified, err := luaMinifier.Minify(source)
			if err != nil {
				return fmt.Errorf("minify %q: %w", relativePath, err)
			}

			if _, err := destination.Write(minified); err != nil {
				return fmt.Errorf("write packaged file %q: %w", zipPath, err)
			}
			return nil
		}

		source, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open source file %q: %w", path, err)
		}

		_, copyErr := io.Copy(destination, source)
		closeErr := source.Close()
		if copyErr != nil {
			return fmt.Errorf("copy source file %q into ZIP: %w", path, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close source file %q: %w", path, closeErr)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("package plugin: %w", err)
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("finalize ZIP: %w", err)
	}

	if err := outputFile.Close(); err != nil {
		return fmt.Errorf("close ZIP file: %w", err)
	}

	complete = true
	return nil
}

func runPackage(args []string, config Config) error {
	flags := flag.NewFlagSet("package", flag.ContinueOnError)

	setUsage(
		flags,
		"kopdev package [options] [plugin-directory]",
		"Create a ZIP archive containing the plugin.",
		"\tplugin-directory  Plugin source directory. Defaults to plugin.source_dir in kopdev.toml.",
		"\tkopdev package\n\tkopdev package ./src\n\tkopdev package --output releases\n\tkopdev package --minify",
	)

	output := flags.String("output", config.Package.OutputDir, "output directory of the ZIP file")
	minifyLua := flags.Bool("minify", config.Package.Minify, "minify Lua files in the ZIP")

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

	options := PackageOptions{
		PluginName: config.Plugin.Name,
		SourceDir:  pluginDirectory,
		OutputDir:  *output,
		Exclude:    config.Package.Exclude,
		Minify:     *minifyLua,
	}
	if err := createPluginZip(options); err != nil {
		return err
	}

	fmt.Printf("Packaged plugin to %s\n", filepath.Join(*output, config.Plugin.Name+".zip"))
	return nil
}
