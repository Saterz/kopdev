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
		return fmt.Errorf("create temporary plugin directory %q: %w", tempOutputDir, err)
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

	rmCommand := exec.Command("ssh", SSHTarget, fmt.Sprintf("test -d %s && rm -rf -- %s && mkdir -- %s", shellQuote(pluginsDir), shellQuote(tempOutputDir), shellQuote(tempOutputDir)))
	rmOutput, err := rmCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("prepare remote directory on %q: %w", SSHTarget, commandError(err, rmOutput))
	}

	scpSource := stagingDir + string(os.PathSeparator) + "."
	scpCommand := exec.Command("scp", "-r", scpSource, fmt.Sprintf("%s:%s/", SSHTarget, tempOutputDir))
	scpOutput, err := scpCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copy plugin to %q: %w", SSHTarget, commandError(err, scpOutput))
	}

	mvCommand := exec.Command("ssh", SSHTarget, fmt.Sprintf("rm -rf -- %s && mv -- %s %s", shellQuote(outputDir), shellQuote(tempOutputDir), shellQuote(outputDir)))
	mvOutput, err := mvCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install plugin on %q: %w", SSHTarget, commandError(err, mvOutput))
	}

	return nil
}

func runDeploy(args []string, config Config) error {
	flags := flag.NewFlagSet("deploy", flag.ContinueOnError)

	setUsage(
		flags,
		"kopdev deploy [options] [plugin-directory]",
		"Deploy the plugin to a KOReader installation over USB or SSH.",
		"\tplugin-directory  Plugin source directory. Defaults to plugin.source_dir in kopdev.toml.",
		"\tkopdev deploy --transport usb --koreader-dir /media/koreader\n\tkopdev deploy --transport ssh --target root@ereader",
	)

	transport := flags.String("transport", config.Deploy.DefaultTransport, "deployment transport method: usb or ssh")

	target := flags.String("target", config.Deploy.SSH.Target, "the remote target when deploying through SSH")

	koreaderDir := flags.String("koreader-dir", "", "KOReader directory")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if *transport != "usb" && *transport != "ssh" {
		if *transport == "" {
			return fmt.Errorf("deployment transport is required; set deploy.default_transport or use --transport")
		}
		return fmt.Errorf("unknown deployment transport %q: expected usb or ssh", *transport)
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
		return fmt.Errorf("KOReader directory %q must be absolute", *koreaderDir)
	}

	positionalArgs := flags.Args()
	if len(positionalArgs) > 1 {
		return fmt.Errorf("usage: kopdev deploy [options] [plugin-directory]")
	}

	pluginName := config.Plugin.Name
	pluginDir := config.Plugin.SourceDir

	if len(positionalArgs) == 1 {
		pluginDir = positionalArgs[0]
	}

	var err error
	switch *transport {
	case "usb":
		err = deployUSB(pluginName, pluginDir, *koreaderDir, config.Deploy.Exclude)
	case "ssh":
		if *target == "" {
			return fmt.Errorf("SSH target is required; set deploy.ssh.target or use --target")
		}
		err = deploySSH(*target, pluginName, pluginDir, *koreaderDir, config.Deploy.Exclude)
	}
	if err != nil {
		return err
	}

	destination := path.Join(*koreaderDir, "plugins", pluginName+".koplugin")
	if *transport == "ssh" {
		destination = *target + ":" + destination
	}
	fmt.Printf("Deployed plugin to %s via %s\n", destination, strings.ToUpper(*transport))
	return nil
}
