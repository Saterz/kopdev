package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func useTemporaryUserConfigDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	original := userConfigDir
	userConfigDir = func() (string, error) { return directory, nil }
	t.Cleanup(func() { userConfigDir = original })
	return directory
}

func TestLoadConfigCombinesProjectAndUserSettings(t *testing.T) {
	userDirectory := useTemporaryUserConfigDir(t)
	if err := writeUserConfig(userConfig{
		Deploy: userDeployConfig{
			DefaultTransport: "ssh",
			SSH: SSHDeployConfig{
				Target:      "root@reader",
				KOReaderDir: "/mnt/koreader",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	projectPath := filepath.Join(t.TempDir(), "kopdev.toml")
	projectContents := `[plugin]
name = "sample"
source_dir = "src"
version = "1.0.0"

[package]
output_dir = "release"
exclude = ["dist"]

[deploy]
exclude = ["tests"]
`
	if err := os.WriteFile(projectPath, []byte(projectContents), 0o600); err != nil {
		t.Fatal(err)
	}

	config, err := loadConfig(projectPath)
	if err != nil {
		t.Fatal(err)
	}
	if config.Plugin.Name != "sample" || config.Package.OutputDir != "release" {
		t.Fatalf("project settings were not loaded: %#v", config)
	}
	if config.Deploy.DefaultTransport != "ssh" ||
		config.Deploy.SSH.Target != "root@reader" ||
		config.Deploy.SSH.KOReaderDir != "/mnt/koreader" {
		t.Fatalf("user settings were not loaded: %#v", config.Deploy)
	}
	if len(config.Deploy.Exclude) != 1 || config.Deploy.Exclude[0] != "tests" {
		t.Fatalf("project deploy exclusions were not preserved: %#v", config.Deploy.Exclude)
	}

	info, err := os.Stat(filepath.Join(userDirectory, "kopdev", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("user config permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestLoadConfigRejectsUserSettingInProjectFile(t *testing.T) {
	useTemporaryUserConfigDir(t)
	projectPath := filepath.Join(t.TempDir(), "kopdev.toml")
	projectContents := `[plugin]
name = "sample"
source_dir = "."

[deploy]
default_transport = "ssh"
`
	if err := os.WriteFile(projectPath, []byte(projectContents), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfig(projectPath)
	if err == nil || !strings.Contains(err.Error(), "user-specific") {
		t.Fatalf("loadConfig error = %v, want user-specific migration error", err)
	}
}

func TestSetUserConfigValueValidatesTransport(t *testing.T) {
	config := userConfig{}
	if err := setUserConfigValue(&config, "deploy.default_transport", "bluetooth"); err == nil {
		t.Fatal("expected invalid transport to fail")
	}
	if err := setUserConfigValue(&config, "deploy.default_transport", "usb"); err != nil {
		t.Fatal(err)
	}
	if config.Deploy.DefaultTransport != "usb" {
		t.Fatalf("transport = %q, want usb", config.Deploy.DefaultTransport)
	}
}
