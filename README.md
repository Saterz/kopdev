# kopdev

A utility program for developing KOReader plugins

## Package a plugin

Create the configured plugin ZIP:

```sh
kopdev package
```

Lua files can be minified only in the generated ZIP. The source files are
left unchanged:

```sh
kopdev package --minify
kopdev package --minify=false
```

Minification can also be enabled in `kopdev.toml`:

```toml
[package]
output_dir = "dist"
minify = true
```

An explicit CLI value overrides the configuration value. Minification is
disabled by default. Kopdev embeds LuaSrcDiet and a Lua 5.1-compatible
GopherLua runtime, so it does not require a separate Lua installation or any
external minifier executable.

## TODO

- Make the package command be able to replace the version name in the _meta.lua file
- Add tools to help develop KOReader patches
- Add the ability to automatically restart KOReader after deploying
- Add the ability to manage changelogs
- Add the ability to include custom file paths which you want to include in the package
- Prevent replacing a previous package when running the package plugin
- Before replacing the previous package in deploy create a backup and delete it later when it succeeds
