# splunk-cli

A pipe-friendly CLI client for the Splunk REST API. Run SPL searches, manage search jobs, and retrieve results directly from the terminal.

[日本語版 README はこちら](README.ja.md)

## Features

- **Synchronous search** — `run` executes a query, waits for completion, and prints results
- **Asynchronous search** — `start` → `status` → `results` for long-running jobs
- **Pipe-friendly** — JSON output composable with `jq`, `json-to-table`, and other tools
- **Flexible authentication** — Token, username/password, env vars, or config file
- **App context** — `--app` flag for app-specific lookups and knowledge objects
- **Ctrl+C handling** — Choose to cancel or background a running job

## Installation

Download a pre-built binary from the [releases page](https://github.com/nlink-jp/splunk-cli/releases).

Or build from source:

```bash
git clone https://github.com/nlink-jp/splunk-cli.git
cd splunk-cli
make build
# Binary: dist/splunk-cli
```

## Quick Start

```bash
# Set credentials
export SPLUNK_HOST="https://your-splunk.example.com:8089"
export SPLUNK_TOKEN="your-token"

# Run a search
splunk-cli run --spl "index=_internal | head 10"

# Pipe to jq
splunk-cli run --spl "index=main | stats count by sourcetype" | jq .

# Read SPL from stdin
cat query.spl | splunk-cli run -f -
```

## Configuration

Copy the example config and set your values:

```bash
mkdir -p ~/.config/splunk-cli
cp config.example.toml ~/.config/splunk-cli/config.toml
chmod 600 ~/.config/splunk-cli/config.toml
```

```toml
# ~/.config/splunk-cli/config.toml
[splunk]
host  = "https://your-splunk.example.com:8089"
token = "your-token"
# app = "search"
# insecure = false
# http_timeout = "30s"
# limit = 0
```

**Priority order (highest first):** CLI flags → environment variables → config file

| Environment variable | Description |
|---|---|
| `SPLUNK_HOST` | Splunk server URL (including port) |
| `SPLUNK_TOKEN` | Bearer token (recommended) |
| `SPLUNK_USER` | Username (basic auth) |
| `SPLUNK_PASSWORD` | Password (basic auth) |
| `SPLUNK_APP` | App context for searches |

## Usage

```
splunk-cli [command]

Commands:
  run         Run a SPL search and print results (synchronous)
  start       Start a SPL search asynchronously and print the SID
  status      Check the status of a search job
  results     Fetch results of a completed search job

Global flags:
  -c, --config string           Config file path (default: ~/.config/splunk-cli/config.toml)
      --host string             Splunk server URL (env: SPLUNK_HOST)
      --token string            Bearer token (env: SPLUNK_TOKEN)
      --user string             Username for basic auth (env: SPLUNK_USER)
      --password string         Password (env: SPLUNK_PASSWORD)
      --app string              App context for searches (env: SPLUNK_APP)
      --owner string            Knowledge object owner (default: nobody)
      --limit int               Max results to return (0 = all)
      --insecure                Skip TLS certificate verification
      --http-timeout duration   Per-request HTTP timeout (e.g. 30s, 2m)
      --debug                   Enable verbose debug logging
  -v, --version                 Print version information
```

### `run` — Synchronous search

```bash
# Search with time range
splunk-cli run --spl "index=_internal" --earliest "-1h" --limit 10

# Read SPL from file
splunk-cli run -f query.spl

# Read SPL from stdin
echo 'index=main | stats count' | splunk-cli run -f -

# With timeout
splunk-cli run --spl "index=main | stats count by host" --timeout 5m
```

| Flag | Description |
|---|---|
| `--spl <string>` | SPL query to execute |
| `-f, --file <path>` | Read SPL from file (use `-` for stdin) |
| `--earliest <time>` | Start time (e.g. `-1h`, `@d`, epoch) |
| `--latest <time>` | End time (e.g. `now`, `@d`, epoch) |
| `--timeout <duration>` | Total job timeout (e.g. `10m`, `1h`) |
| `--limit <int>` | Max results (0 = all) |
| `--silent` | Suppress progress messages |

> **Ctrl+C**: during `run`, you can choose to cancel the job or let it continue in the background.

### `start` — Asynchronous search

```bash
JOB_ID=$(splunk-cli start --spl "index=main | stats count by sourcetype")
echo "Started: $JOB_ID"
```

### `status` — Check job status

```bash
splunk-cli status --sid "$JOB_ID"
```

### `results` — Fetch job results

```bash
splunk-cli results --sid "$JOB_ID" --limit 50 --silent | jq .
```

## Building

```bash
make build            # Current platform → dist/splunk-cli
make build-all        # All platforms → dist/
make test             # Unit tests
make check            # vet → lint → test → build
make integration-test # Integration tests (requires Podman + Splunk container)
make splunk-down      # Stop Splunk test container
make clean            # Remove dist/
```

See [BUILD.md](BUILD.md) for detailed build and integration test instructions.

## License

MIT License. See [LICENSE](LICENSE).
