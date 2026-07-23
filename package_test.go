package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const realisticLua = `-- plugin comment
local greeting = "hello -- this is not a comment"

local function greet(name)
    return greeting .. ", " .. name
end

return {
    greet = greet,
}
`

func TestCreatePluginZipWithoutMinifyPreservesLua(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceDir := filepath.Join(root, "src")
	outputDir := filepath.Join(root, "dist")
	writeTestFile(t, filepath.Join(sourceDir, "main.lua"), []byte(realisticLua))

	err := createPluginZip(PackageOptions{
		PluginName: "sample",
		SourceDir:  sourceDir,
		OutputDir:  outputDir,
	})
	if err != nil {
		t.Fatalf("createPluginZip() error = %v", err)
	}

	files := readTestZip(t, filepath.Join(outputDir, "sample.zip"))
	if got := files["sample.koplugin/main.lua"]; !bytes.Equal(got, []byte(realisticLua)) {
		t.Errorf("packaged Lua differs from source:\ngot:  %q\nwant: %q", got, realisticLua)
	}
}

func TestCreatePluginZipWithMinify(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceDir := filepath.Join(root, "src")
	outputDir := filepath.Join(root, "dist")
	mainPath := filepath.Join(sourceDir, "main.lua")
	secondPath := filepath.Join(sourceDir, "lib", "second.lua")
	assetPath := filepath.Join(sourceDir, "assets", "data.bin")
	excludedPath := filepath.Join(sourceDir, "ignored.lua")

	mainSource := []byte(realisticLua)
	secondSource := []byte("-- another comment\nreturn { value = 42 }\n")
	assetSource := []byte{0, 1, 2, '\n', 255}
	excludedSource := []byte("this is invalid Lua and must not be processed")

	writeTestFile(t, mainPath, mainSource)
	writeTestFile(t, secondPath, secondSource)
	writeTestFile(t, assetPath, assetSource)
	writeTestFile(t, excludedPath, excludedSource)

	err := createPluginZip(PackageOptions{
		PluginName: "sample",
		SourceDir:  sourceDir,
		OutputDir:  outputDir,
		Exclude:    []string{"ignored.lua"},
		Minify:     true,
	})
	if err != nil {
		t.Fatalf("createPluginZip() error = %v", err)
	}

	files := readTestZip(t, filepath.Join(outputDir, "sample.zip"))
	mainResult := files["sample.koplugin/main.lua"]
	secondResult := files["sample.koplugin/lib/second.lua"]

	if bytes.Contains(mainResult, []byte("plugin comment")) {
		t.Errorf("packaged main.lua retained comment:\n%s", mainResult)
	}
	if !bytes.Contains(mainResult, []byte(`"hello -- this is not a comment"`)) {
		t.Errorf("packaged main.lua changed string contents:\n%s", mainResult)
	}
	if len(mainResult) >= len(mainSource) {
		t.Errorf("packaged main.lua length = %d, want less than %d", len(mainResult), len(mainSource))
	}
	if bytes.Contains(secondResult, []byte("another comment")) {
		t.Errorf("packaged second.lua retained comment:\n%s", secondResult)
	}
	if !bytes.Equal(files["sample.koplugin/assets/data.bin"], assetSource) {
		t.Errorf("packaged non-Lua file changed")
	}
	if _, ok := files["sample.koplugin/ignored.lua"]; ok {
		t.Errorf("excluded Lua file was packaged")
	}

	assertTestFileEquals(t, mainPath, mainSource)
	assertTestFileEquals(t, secondPath, secondSource)
	assertTestFileEquals(t, assetPath, assetSource)
	assertTestFileEquals(t, excludedPath, excludedSource)
}

func TestCreatePluginZipMinifyFailureRemovesArchive(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceDir := filepath.Join(root, "src")
	outputDir := filepath.Join(root, "dist")
	writeTestFile(t, filepath.Join(sourceDir, "a-valid.lua"), []byte("return true\n"))
	writeTestFile(t, filepath.Join(sourceDir, "z-invalid.lua"), []byte("local function broken(\n"))

	err := createPluginZip(PackageOptions{
		PluginName: "sample",
		SourceDir:  sourceDir,
		OutputDir:  outputDir,
		Minify:     true,
	})
	if err == nil {
		t.Fatal("createPluginZip() error = nil, want malformed Lua error")
	}
	if !strings.Contains(err.Error(), `minify "z-invalid.lua": LuaSrcDiet:`) {
		t.Errorf("createPluginZip() error = %q, want file and LuaSrcDiet context", err)
	}
	if _, statErr := os.Stat(filepath.Join(outputDir, "sample.zip")); !os.IsNotExist(statErr) {
		t.Errorf("partial archive still exists; os.Stat() error = %v", statErr)
	}
}

func TestRunPackageMinifyPrecedence(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sourceDir := filepath.Join(root, "src")
	writeTestFile(t, filepath.Join(sourceDir, "main.lua"), []byte(realisticLua))

	tests := []struct {
		name         string
		configMinify bool
		args         []string
		wantMinified bool
	}{
		{
			name:         "configuration enables minification",
			configMinify: true,
			wantMinified: true,
		},
		{
			name:         "explicit CLI true overrides configuration",
			args:         []string{"--minify"},
			wantMinified: true,
		},
		{
			name:         "explicit CLI false overrides configuration",
			configMinify: true,
			args:         []string{"--minify=false"},
			wantMinified: false,
		},
		{
			name:         "default is false",
			wantMinified: false,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			outputDir := filepath.Join(root, strings.ReplaceAll(test.name, " ", "-"))
			config := defaultConfig()
			config.Plugin.Name = "sample"
			config.Plugin.SourceDir = sourceDir
			config.Package.OutputDir = outputDir
			config.Package.Minify = test.configMinify

			if err := runPackage(test.args, config); err != nil {
				t.Fatalf("runPackage() error = %v", err)
			}

			files := readTestZip(t, filepath.Join(outputDir, "sample.zip"))
			result := files["sample.koplugin/main.lua"]
			gotMinified := !bytes.Contains(result, []byte("plugin comment"))
			if gotMinified != test.wantMinified {
				t.Errorf("runPackage() minified = %t, want %t; output:\n%s", gotMinified, test.wantMinified, result)
			}
		})
	}
}

func writeTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertTestFileEquals(t *testing.T, path string, want []byte) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("source file %q changed:\ngot:  %q\nwant: %q", path, got, want)
	}
}

func readTestZip(t *testing.T, path string) map[string][]byte {
	t.Helper()

	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("zip.OpenReader(%q) error = %v", path, err)
	}
	defer reader.Close()

	files := make(map[string][]byte, len(reader.File))
	for _, file := range reader.File {
		source, err := file.Open()
		if err != nil {
			t.Fatalf("open ZIP entry %q error = %v", file.Name, err)
		}
		contents, err := io.ReadAll(source)
		if err != nil {
			source.Close()
			t.Fatalf("read ZIP entry %q error = %v", file.Name, err)
		}
		if err := source.Close(); err != nil {
			t.Fatalf("close ZIP entry %q error = %v", file.Name, err)
		}
		files[file.Name] = contents
	}
	return files
}
