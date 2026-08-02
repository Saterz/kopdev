# kopdev

A utility program for developing KOReader plugins.

## Configuration

Kopdev separates project settings from settings that belong to the current user.

Each plugin project keeps `kopdev.toml` in its root. It contains the plugin name,
source directory, version, package options, and deployment exclusions:

```toml
[plugin]
name = "my-plugin"
source_dir = "src"
version = "0.1.0"

[package]
output_dir = "dist"
exclude = [".git", "dist", "kopdev.toml"]

[deploy]
exclude = []
```

Deployment destinations belong to the developer and are stored in the operating
system's user config directory. Manage them without editing the file directly:

```sh
kopdev config set deploy.default_transport ssh
kopdev config set deploy.ssh.target root@ereader
kopdev config set deploy.ssh.koreader_dir /mnt/onboard/.adds/koreader
kopdev config list
```

Use `kopdev config path` to print the user config file location, `kopdev config
get <key>` to read one value, and `kopdev config unset <key>` to clear one.

Existing projects must remove `deploy.default_transport`, `deploy.usb`, and
`deploy.ssh` from `kopdev.toml` and set those values with `kopdev config set`.

## TODO

- Make the package command be able to replace the version name in the _meta.lua file
- Add the ability to minify the code to lower the plugin size
- Add tools to help develop KOReader patches
- Add the ability to automatically restart KOReader after deploying
- Add the ability to manage changelogs
- Add the ability to include custom file paths which you want to include in the package
- Prevent replacing a previous package when running the package plugin
- Before replacing the previous package in deploy create a backup and delete it later when it succeeds
