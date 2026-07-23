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
	Minify    bool     `toml:"minify"`
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

var version string = "development"

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
	config := defaultConfig()

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

func defaultConfig() Config {
	return Config{
		Package: PackageConfig{
			OutputDir: "dist",
		},
	}
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

	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return command(args, defaultConfig())
		}
	}

	config, err := loadConfig(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	return command(args, config)
}

func run(args []string) error {
	if len(args) == 0 {
		runHelp(nil)
		return errors.New("no command given; run \"kopdev help\" for usage")
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])

	case "package":
		return runWithConfig(args[1:], runPackage)

	case "deploy":
		return runWithConfig(args[1:], runDeploy)

	case "version", "--version", "-v":
		if len(args) != 1 {
			return errors.New("usage: kopdev version")
		}
		fmt.Printf("Kopdev %s\n", version)
		return nil

	case "help", "--help", "-h":
		return runHelp(args[1:])
	default:
		return fmt.Errorf("unknown command %q; run \"kopdev help\" for usage", args[0])
	}

}

func runHelp(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: kopdev help [command]")
	}

	if len(args) == 1 {
		switch args[0] {
		case "init":
			return runInit([]string{"--help"})
		case "package":
			return runPackage([]string{"--help"}, defaultConfig())
		case "deploy":
			return runDeploy([]string{"--help"}, defaultConfig())
		default:
			return fmt.Errorf("unknown help topic %q; run \"kopdev help\" for available commands", args[0])
		}
	}

	fmt.Println("Kopdev is a utility program to help develop KOReader plugins\n\n" +
		"Usage:\n\n\tkopdev <command> [arguments]\n\n" +
		"The commands are:\n\n" +
		"\tinit         create a KOReader plugin project from a template\n" +
		"\tpackage      package a KOReader plugin into a ZIP file\n" +
		"\tdeploy       deploys the plugin to a remote device\n" +
		"\tversion      print the kopdev version\n\n" +
		"Use \"kopdev help <command>\" for more information about a command.")

	return nil
}
