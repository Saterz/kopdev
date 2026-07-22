package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Plugin  PluginConfig  `toml:"plugin"`
	Package PackageConfig `toml:"package"`
	Deploy  DeployConfig  `toml:"deploy"`
}

type PluginConfig struct {
	Name      string `toml:"name"`
	SourceDir string `toml:"source_dir"`
	Version   string `toml:"version"`
}

type PackageConfig struct {
	OutputDir string   `toml:"output_dir"`
	Exclude   []string `toml:"exclude"`
}

type DeployConfig struct {
	DefaultTransport string          `toml:"default_transport"`
	USB              USBDeployConfig `toml:"usb"`
	SSH              SSHDeployConfig `toml:"ssh"`
	Exclude          []string        `toml:"exclude"`
}

type USBDeployConfig struct {
	KOReaderDir string `toml:"koreader_dir"`
}

type SSHDeployConfig struct {
	Target      string `toml:"target"`
	KOReaderDir string `toml:"koreader_dir"`
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
		return Config{}, fmt.Errorf("read %q: %w", path, err)
	}

	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("parse %q: %w", path, err)
	}

	if err := validatePluginName(config.Plugin.Name); err != nil {
		return Config{}, err
	}

	return config, nil
}

func isSymlink(entry fs.DirEntry) bool {
	return entry.Type()&os.ModeSymlink != 0
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
		return errors.New("no command given; run \"kopdev help\" for usage")
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])

	case "package":
		return runWithConfig(args[1:], runPackage)

	case "deploy":
		return runWithConfig(args[1:], runDeploy)

	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q; run \"kopdev help\" for usage", args[0])
	}

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
