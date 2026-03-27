# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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