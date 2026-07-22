package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func deployUSB(pluginName string, sourceDir string, KOReaderDir string, exclude []string) error {
	sourceDir = filepath.Clean(sourceDir)
	KOReaderDir = filepath.Clean(KOReaderDir)

	pluginsDir := filepath.Join(KOReaderDir, "plugins")

	pluginFolderName := pluginName + ".koplugin"

	outputDir := filepath.Join(pluginsDir, pluginFolderName)
	tempOutputDir := filepath.Join(outputDir + ".kopdev-tmp")
	if err := os.RemoveAll(tempOutputDir); err != nil {
		return fmt.Errorf("remove stale temporary directory: %w", err)
	}
	if err := os.Mkdir(tempOutputDir, 0o755); err != nil {
		os.RemoveAll(tempOutputDir)
		return err
	}
	defer os.RemoveAll(tempOutputDir)

	if err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
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

		tempDestinationPath := filepath.Join(tempOutputDir, relativePath)

		if err := os.MkdirAll(filepath.Dir(tempDestinationPath), 0o755); err != nil {
			return err
		}

		destination, err := os.Create(tempDestinationPath)
		if err != nil {
			return err
		}

		source, err := os.Open(path)
		if err != nil {
			destination.Close()
			return fmt.Errorf("read file: %w", err)
		}

		_, err = io.Copy(destination, source)
		if err != nil {
			destination.Close()
			source.Close()
			return err
		}

		if err := destination.Close(); err != nil {
			source.Close()
			return err
		}

		err = source.Close()

		return err
	}); err != nil {
		return fmt.Errorf("deploy plugin: %w", err)
	}

	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("remove existing plugin: %w", err)
	}

	if err := os.Rename(tempOutputDir, outputDir); err != nil {
		return fmt.Errorf("install plugin: %w", err)
	}

	return nil
}

func deploySSH(pluginName string, sourceDir string, outputDir string, exclude []string) error {
	return nil
}

func runDeploy(args []string, config Config) error {
	flags := flag.NewFlagSet("deploy", flag.ContinueOnError)

	transport := flags.String("transport", config.Deploy.DefaultTransport, "deployment transport method: usb or ssh")

	koreaderDir := flags.String("koreader-dir", "", "remote KOReader directory")

	if err := flags.Parse(args); err != nil {
		return err
	}

	if *koreaderDir == "" {
		switch *transport {
		case "usb":
			*koreaderDir = config.Deploy.USB.KOReaderDir
		case "ssh":
			*koreaderDir = config.Deploy.SSH.KOReaderDir
		}
	}

	positionalArgs := flags.Args()

	pluginName := config.Plugin.Name
	pluginDir := config.Plugin.SourceDir

	if len(positionalArgs) == 1 {
		pluginDir = positionalArgs[0]
	}

	switch *transport {
	case "usb":
		return deployUSB(pluginName, pluginDir, *koreaderDir, config.Deploy.Exclude)
	case "ssh":
		return deploySSH(pluginName, pluginDir, *koreaderDir, config.Deploy.Exclude)
	default:
		return fmt.Errorf("unknown deployment transport %q: expected usb or ssh", *transport)
	}
}
