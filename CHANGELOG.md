# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.2.0] - 2026-07-12

### Removed

- **darwin/amd64 (Intel) and darwin universal (lipo) pre-built binaries.**
  macOS releases now ship a single **arm64** archive, per the org-wide policy
  (darwin is Apple-Silicon only; no Intel and no universal binaries — the
  universal binary doubled the download size). Intel Mac users can build from
  source.

### Changed

- **Linux release archives are now `.tar.gz`** (darwin/windows remain `.zip`),
  per `nlink-jp/.github` CONVENTIONS.md §Release Archive Standard.
- **`LICENSE` is now bundled** in every release archive alongside `README.md`.
- **darwin code-signature identifier** is now the canonical `splunk-cli`
  (was `splunk-cli-darwin-arm64`), set via `codesign -i` so it stays stable
  after the archived binary is renamed to its canonical name.

No change to the binary's behaviour — a packaging / build-config release.

## [2.1.0] - 2026-05-23

### Added

- **`--prepend` flag and `prepend` config field** for choosing how SPL
  is wrapped before submission. Three modes:
  - `pipe-only` (default, historical behavior): prepend `search ` unless
    the SPL starts with `|`.
  - `auto`: also skip the prefix when the SPL already starts with the
    `search` command (followed by whitespace or end-of-string). Avoids
    the doubled-`search` artifact when a user pastes `search index=foo`
    from Splunk Web. Does not detect macros that expand to a leading
    command — use `off` for that.
  - `off`: never prepend; the caller supplies a complete SPL.

  Precedence: `--prepend` CLI flag > `[splunk] prepend` in config file
  > built-in default (`pipe-only`). Default is unchanged from prior
  versions, so existing workflows are unaffected. Addresses
  https://github.com/nlink-jp/splunk-cli/issues/4.

### Changed

- The auto-prepend logic moves into the new `internal/spl` package as
  a pure `Wrap(spl, mode)` helper, and the Splunk client now delegates
  to it instead of inlining the rule. No observable change at the
  default mode.

## [2.0.4] - 2026-05-22

### Added

- **`package` Makefile target.** Builds all 5 platforms (plus the
  darwin-universal binary when `lipo` is available), signs every
  darwin variant, zips each with README.md using versioned naming
  (`splunk-cli-vX.Y.Z-<os>-<arch>.zip`), and notarizes the darwin
  zips. v2.0.3 was tagged without binary assets; v2.0.4 restores
  the binary release flow (last seen in v2.0.2) with proper
  signing and notarization.

### Changed

- **Darwin releases are now Developer ID signed and Apple-notarized.**
  All three darwin variants — `darwin-amd64`, `darwin-arm64`, and
  `darwin-universal` — carry full Apple Developer ID Application
  signatures and notarization tickets from Apple. End users on
  macOS no longer need to bypass Gatekeeper with right-click → Open
  or `xattr -d com.apple.quarantine` on first launch; local users
  who place `splunk-cli` under Dropbox-synced (or any other
  FileProvider-managed) paths are no longer killed by macOS's
  ad-hoc + provenance distrust policy. Pipeline:
  `scripts/codesign-darwin.sh` + `scripts/notarize-darwin.sh`,
  driven by `make package`. Adopts the org-wide convention in
  `nlink-jp/.github` CONVENTIONS.md §Code Signing.
- **Release zip filenames now embed the version**
  (`splunk-cli-vX.Y.Z-<os>-<arch>.zip`), aligning with sibling
  cli-series tools. v2.0.2 assets used version-less names.

No behaviour change to the binary itself — feature-wise this is
identical to v2.0.3.

## [2.0.3] - 2026-03-31

### Fixed
- Skip config file permission check on Windows/NTFS (always reports 0666 regardless of ACLs)
- Document NTFS ACL-based alternative for securing config files on Windows

## [2.0.2] - 2026-03-27

### Added

- Integration tests against a live Splunk instance (`//go:build integration`).
  Covers: full search lifecycle, limit, empty results, cancel, invalid SPL,
  and search-prefix behaviour.
- `scripts/splunk-up.sh` / `scripts/splunk-down.sh`: start and stop a
  `splunk/splunk` container via Podman for local integration testing.
- `make integration-test` / `make splunk-up` / `make splunk-down` targets.

### Documentation

- `BUILD.md` rewritten with current build, test, and release instructions.
- `CLAUDE.md` testing section expanded with integration-test targets.
- `README.md` development section updated with current Makefile targets.

## [2.0.1] - 2026-03-27

### Fixed

- Fixed a resource leak where `resp.Body` was deferred inside the results
  pagination loop, preventing response bodies from being closed until the
  entire `Results()` call returned. Extracted `fetchResultsPage()` helper.
- Fixed empty results marshalling as `{"results": null}` instead of
  `{"results": []}` when a completed job has zero results.
- Eliminated a redundant `GetJobStatus` API call: `Results()` previously
  fetched job status internally even though the caller had already done so.
  The function now accepts `totalResults int` from the caller.

## [2.0.0] - 2026-03-27

### Breaking

- Config format changed from JSON (`config.json`) to TOML (`config.toml`).
  Rename `~/.config/splunk-cli/config.json` to `config.toml` and update the
  format — see `config.example.toml` for the new structure.
- Go module path changed to `github.com/nlink-jp/splunk-cli`.

### Changed

- Migrated from `nlink-jp` organization (transferred from `magifd2/splunk-cli`).
- CLI framework replaced with [Cobra](https://github.com/spf13/cobra);
  all commands and flags remain the same.
- Splunk client moved to `internal/client`; config loading moved to `internal/config`.
- Added config file permission check: warns if file is readable by group or others.
- Added warning when sending a bearer token over unencrypted HTTP.
- Makefile aligned with cli-series conventions (`check`, `build-all` targets).

### Internal

- Added unit tests for config loading and Splunk API client.

## [1.4.0] - 2025-08-28

### Changed

- Implemented pagination for result fetching to correctly handle large result sets that exceed the API's single-request limit. This ensures that `--limit 0` fetches all results and that `--limit` values greater than 50,000 are respected.

## [1.3.0] - 2025-08-28

### Added

- Added a `--limit` flag to the `run` and `results` commands to control the maximum number of results returned.
- Added a `limit` field to the `config.json` file to allow setting a default result limit.

### Changed

- The default behavior for result fetching is now to return all results (`limit=0`) unless specified otherwise by the `--limit` flag or in the config file.

### Fixed

- Fixed a display issue where the "Waiting for job to complete..." message was not printed on a new line.

## [1.2.1] - 2025-08-18

### Fixed

- Fixed an issue where the version information was not correctly embedded in the binary during the `make` build process. The build script now correctly links the Git tag, commit hash, and build date.

## [1.2.0] - 2025-08-14

### Changed

- **Major Refactoring**: The entire codebase has been refactored for better modularity, testability, and maintainability.
  - Core Splunk API interaction logic has been extracted into a new `splunk` package.
  - Command-line interface (CLI) logic has been separated into a new `cmd` package, with each command in its own file.
  - The main application entrypoint (`splunk-cli.go`) is now significantly simplified.

## [1.1.0] - 2025-08-12

### Added

- Added a global `--config` flag to specify a custom configuration file path, overriding the default `~/.config/splunk-cli/config.json`.

## [1.0.0] - 2025-08-05

### Added

- **Initial Release** of `splunk-cli`.
- Core functionalities: `run`, `start`, `status`, `results` commands to interact with Splunk's REST API.
- Flexible authentication via config file, environment variables, or command-line flags.
- Support for reading SPL queries from files or standard input.
- Asynchronous job handling with job cancellation support (`Ctrl+C`).
- App context support for searches (`--app` flag).
- Makefile for simplified building, testing, linting, and vulnerability scanning.
- Cross-platform build support for macOS (Universal), Linux (amd64), and Windows (amd64).
- Version information embedded in the binary (`--version` flag).
- `README.md` and `LICENSE` (MIT) for project documentation.
- `CHANGELOG.md` to track project changes.
- Japanese README (`README.ja.md`).

### Changed

- Switched build system from a shell script (`build.sh`) to a `Makefile`.

### Fixed

- N/A (Initial Release)