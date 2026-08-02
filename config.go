package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
)

type projectConfig struct {
	Plugin  PluginConfig        `mapstructure:"plugin"`
	Package PackageConfig       `mapstructure:"package"`
	Deploy  projectDeployConfig `mapstructure:"deploy"`
}

type projectDeployConfig struct {
	Exclude []string `mapstructure:"exclude"`
}

type userConfig struct {
	Deploy userDeployConfig `mapstructure:"deploy"`
}

type userDeployConfig struct {
	DefaultTransport string          `mapstructure:"default_transport"`
	USB              USBDeployConfig `mapstructure:"usb"`
	SSH              SSHDeployConfig `mapstructure:"ssh"`
}

var userConfigDir = os.UserConfigDir

var userConfigKeys = []string{
	"deploy.default_transport",
	"deploy.usb.koreader_dir",
	"deploy.ssh.target",
	"deploy.ssh.koreader_dir",
}

func defaultConfig() Config {
	return Config{
		Package: PackageConfig{
			OutputDir: "dist",
		},
	}
}

func getUserConfigPath() (string, error) {
	directory, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(directory, "kopdev", "config.toml"), nil
}

func loadProjectConfig(path string) (projectConfig, error) {
	config := projectConfig{
		Package: PackageConfig{OutputDir: "dist"},
	}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")
	v.SetDefault("package.output_dir", "dist")

	if err := v.ReadInConfig(); err != nil {
		return projectConfig{}, fmt.Errorf("read %q: %w", path, err)
	}

	if err := v.UnmarshalExact(&config); err != nil {
		return projectConfig{}, fmt.Errorf("parse %q: %w", path, err)
	}

	if err := validatePluginName(config.Plugin.Name); err != nil {
		return projectConfig{}, err
	}
	return config, nil
}

func loadUserConfig() (userConfig, error) {
	path, err := getUserConfigPath()
	if err != nil {
		return userConfig{}, err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return userConfig{}, nil
	} else if err != nil {
		return userConfig{}, fmt.Errorf("inspect %q: %w", path, err)
	}

	config := userConfig{}
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")
	if err := v.ReadInConfig(); err != nil {
		return userConfig{}, fmt.Errorf("read %q: %w", path, err)
	}
	if err := v.UnmarshalExact(&config); err != nil {
		return userConfig{}, fmt.Errorf("parse %q: %w", path, err)
	}
	return config, nil
}

func loadConfig(path string) (Config, error) {
	project, err := loadProjectConfig(path)
	if err != nil {
		return Config{}, err
	}
	user, err := loadUserConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Plugin:  project.Plugin,
		Package: project.Package,
		Deploy: DeployConfig{
			DefaultTransport: user.Deploy.DefaultTransport,
			USB:              user.Deploy.USB,
			SSH:              user.Deploy.SSH,
			Exclude:          project.Deploy.Exclude,
		},
	}, nil
}

func writeUserConfig(config userConfig) error {
	path, err := getUserConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create user config directory: %w", err)
	}

	v := viper.New()
	v.SetConfigType("toml")
	v.Set("deploy.default_transport", config.Deploy.DefaultTransport)
	v.Set("deploy.usb.koreader_dir", config.Deploy.USB.KOReaderDir)
	v.Set("deploy.ssh.target", config.Deploy.SSH.Target)
	v.Set("deploy.ssh.koreader_dir", config.Deploy.SSH.KOReaderDir)
	if err := v.WriteConfigAs(path); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("set permissions on %q: %w", path, err)
	}
	return nil
}

func userConfigValue(config userConfig, key string) (string, error) {
	switch key {
	case "deploy.default_transport":
		return config.Deploy.DefaultTransport, nil
	case "deploy.usb.koreader_dir":
		return config.Deploy.USB.KOReaderDir, nil
	case "deploy.ssh.target":
		return config.Deploy.SSH.Target, nil
	case "deploy.ssh.koreader_dir":
		return config.Deploy.SSH.KOReaderDir, nil
	default:
		return "", fmt.Errorf("unknown user config key %q", key)
	}
}

func setUserConfigValue(config *userConfig, key, value string) error {
	switch key {
	case "deploy.default_transport":
		if value != "" && value != "usb" && value != "ssh" {
			return fmt.Errorf("invalid transport %q: expected usb or ssh", value)
		}
		config.Deploy.DefaultTransport = value
	case "deploy.usb.koreader_dir":
		config.Deploy.USB.KOReaderDir = value
	case "deploy.ssh.target":
		config.Deploy.SSH.Target = value
	case "deploy.ssh.koreader_dir":
		config.Deploy.SSH.KOReaderDir = value
	default:
		return fmt.Errorf("unknown user config key %q", key)
	}
	return nil
}

func printConfigUsage() {
	fmt.Println("Usage:\n\n" +
		"Manage global configs\n\n" +
		"\tkopdev config list\n" +
		"\tkopdev config get <key>\n" +
		"\tkopdev config set <key> <value>\n" +
		"\tkopdev config unset <key>\n" +
		"\tkopdev config edit\n" +
		"\tkopdev config path")
}

func runConfig(args []string) error {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "-h" || args[0] == "--help")) {
		printConfigUsage()
		return nil
	}

	if args[0] == "path" {
		if len(args) != 1 {
			return errors.New("usage: kopdev config path")
		}
		path, err := getUserConfigPath()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return nil
	}

	config, err := loadUserConfig()
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		if len(args) != 1 {
			return errors.New("usage: kopdev config list")
		}
		for _, key := range userConfigKeys {
			value, _ := userConfigValue(config, key)
			fmt.Printf("%s=%s\n", key, value)
		}
	case "get":
		if len(args) != 2 {
			return errors.New("usage: kopdev config get <key>")
		}
		value, err := userConfigValue(config, args[1])
		if err != nil {
			return err
		}
		fmt.Println(value)
	case "set":
		if len(args) != 3 {
			return errors.New("usage: kopdev config set <key> <value>")
		}
		if err := setUserConfigValue(&config, args[1], args[2]); err != nil {
			return err
		}
		if err := writeUserConfig(config); err != nil {
			return err
		}
		fmt.Printf("Set %s\n", args[1])
	case "unset":
		if len(args) != 2 {
			return errors.New("usage: kopdev config unset <key>")
		}
		if err := setUserConfigValue(&config, args[1], ""); err != nil {
			return err
		}
		if err := writeUserConfig(config); err != nil {
			return err
		}
		fmt.Printf("Unset %s\n", args[1])
	case "edit":
		path, err := getUserConfigPath()
		if err != nil {
			return err
		}
		command := exec.Command("vim", path)
		command.Stdin = os.Stdin
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("edit config: %w", err)
		}
	default:
		return fmt.Errorf("unknown config command %q; run \"kopdev help config\" for usage", args[0])
	}
	return nil
}
