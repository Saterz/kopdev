# kopdev

A utility CLI tool for developing KOReader plugins.

## Usage

*TBD* - This plugin is still in its early state and a lot of things must be fixed/improve/added before its first initial release. Major (and potentially breaking) changes can be made without warning between commits. **Use at your own risk!!!!**

## Tests

Before KOReader starts, `kopdev test` copies `specs/fixtures/` into its
temporary `KO_HOME`. Put files in the same relative paths KOReader expects.
For example, a plugin can supply test-only settings at
`specs/fixtures/settings/<plugin>_settings.lua`.

## TODO

- [ ] FIND A NEW NAME. The current one is too close to KOReader's `kodev` command.
- [ ] Make the package command be able to replace the version name in the _meta.lua file
- [x] Add the ability to minify the code to lower the plugin size
- [ ] Add tools to help develop KOReader patches
- [ ] Add the ability to automatically restart KOReader after deploying
- [ ] Add the ability to manage plugin version changelogs
- [ ] Improve the project configuration to facilitate the addition and exclusion of files to be package
- [ ] Prevent replacing a previous package when running the package plugin
- [ ] Before replacing the previous package in deploy create a backup and delete it later when it succeeds
