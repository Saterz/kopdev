package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
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

	if err := copyFilteredFiles(sourceDir, tempOutputDir, exclude); err != nil {
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

func deploySSH(SSHTarget string, pluginName string, sourceDir string, KOReaderDir string, exclude []string) error {
	stagingDir, err := os.MkdirTemp("", "kopdev-deploy-*")
	if err != nil {
		return fmt.Errorf("create deploy staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	if err := copyFilteredFiles(sourceDir, stagingDir, exclude); err != nil {
		return fmt.Errorf("prepare deployment: %w", err)
	}

	KOReaderDir = path.Clean(KOReaderDir)

	pluginsDir := path.Join(KOReaderDir, "plugins")

	pluginFolderName := pluginName + ".koplugin"
	outputDir := path.Join(pluginsDir, pluginFolderName)
	tempOutputDir := outputDir + ".kopdev-temp"

	rmCommand := exec.Command("ssh", SSHTarget, fmt.Sprintf("test -d -- %s && rm -rf -- %s && mkdir -- %s", shellQuote(pluginsDir), shellQuote(tempOutputDir), shellQuote(tempOutputDir)))
	rmOutput, err := rmCommand.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(rmOutput))
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}

	scpSource := stagingDir + string(os.PathSeparator) + "."
	scpCommand := exec.Command("scp", "-r", scpSource, fmt.Sprintf("%s:%s/", SSHTarget, tempOutputDir))
	scpOutput, err := scpCommand.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(scpOutput))
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}

	mvCommand := exec.Command("ssh", SSHTarget, fmt.Sprintf("rm -rf -- %s && mv -- %s %s", shellQuote(outputDir), shellQuote(tempOutputDir), shellQuote(outputDir)))
	mvOutput, err := mvCommand.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(mvOutput))
		if message != "" {
			return fmt.Errorf("%w: %s", err, message)
		}
		return err
	}

	return nil
}

func runDeploy(args []string, config Config) error {
	flags := flag.NewFlagSet("deploy", flag.ContinueOnError)

	transport := flags.String("transport", config.Deploy.DefaultTransport, "deployment transport method: usb or ssh")

	target := flags.String("target", config.Deploy.SSH.Target, "the remote target when deploying through SSH")

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

	if !path.IsAbs(*koreaderDir) {
		return errors.New("remote KOReader directory must be absolute")
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
		return deploySSH(*target, pluginName, pluginDir, *koreaderDir, config.Deploy.Exclude)
	default:
		return fmt.Errorf("unknown deployment transport %q: expected usb or ssh", *transport)
	}
}
