package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

type Config struct {
	Plugin  PluginConfig
	Package PackageConfig
	Deploy  DeployConfig
}

type PluginConfig struct {
	Name      string `mapstructure:"name"`
	SourceDir string `mapstructure:"source_dir"`
	Version   string `mapstructure:"version"`
}

type PackageConfig struct {
	OutputDir string   `mapstructure:"output_dir"`
	Exclude   []string `mapstructure:"exclude"`
}

type DeployConfig struct {
	DefaultTransport string
	USB              USBDeployConfig
	SSH              SSHDeployConfig
	Exclude          []string
}

type USBDeployConfig struct {
	KOReaderDir string `mapstructure:"koreader_dir"`
}

type SSHDeployConfig struct {
	Target      string `mapstructure:"target"`
	KOReaderDir string `mapstructure:"koreader_dir"`
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
		return nil
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "package":
		return runWithConfig(args[1:], runPackage)
	case "deploy":
		return runWithConfig(args[1:], runDeploy)
	case "config":
		return runConfig(args[1:])
	case "test":
		return runWithConfig(args[1:], runTest)
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
		case "config":
			return runConfig([]string{"--help"})
		default:
			return fmt.Errorf("unknown help topic %q; run \"kopdev help\" for available commands", args[0])
		}
	}

	fmt.Println("Kopdev is a utility program to help develop KOReader plugins\n\n" +
		"Usage:\n\n\tkopdev <command> [arguments]\n\n" +
		"The commands are:\n\n" +
		"\tinit         create a KOReader plugin project from a template\n" +
		"\tpackage      package a KOReader plugin into a ZIP file\n" +
		"\tdeploy       deploy the plugin to a remote device\n" +
		"\tconfig       manage user-wide configuration\n" +
		"\tversion      print the kopdev version\n\n" +
		"Use \"kopdev help <command>\" for more information about a command.")

	return nil
}
