package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigPackageMinify(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "kopdev.toml")
	contents := []byte(`[plugin]
name = "sample"
source_dir = "src"

[package]
output_dir = "dist"
minify = true
`)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if !config.Package.Minify {
		t.Errorf("config.Package.Minify = false, want true")
	}
}

func TestDefaultConfigDisablesMinify(t *testing.T) {
	t.Parallel()

	if defaultConfig().Package.Minify {
		t.Errorf("defaultConfig().Package.Minify = true, want false")
	}
}
