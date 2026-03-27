# Development Guide

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Build and test |
| golangci-lint | latest | Lint (`make lint`) |
| Podman | 4.0+ | Integration tests only |
| Python 3 | any | `scripts/splunk-up.sh` helper |
| curl | any | `scripts/splunk-up.sh` helper |

## Build

```bash
make build       # current platform → ./splunk-cli
make build-all   # cross-compile → dist/
make clean       # remove artifacts
```

Cross-compiled targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64,
darwin universal (lipo), windows/amd64.

## Unit Tests

```bash
make test    # go test ./...
make check   # vet → lint → test → build (full quality gate)
```

Unit tests use only the standard library and mock HTTP servers — no external
services required.

## Integration Tests

Integration tests run against a real Splunk instance.  They use the
`integration` build tag so they are excluded from `make test` / `make check`.

### Requirements

- **Podman** with a running machine (`podman machine start`)
- First run downloads `splunk/splunk:9.4` (~1.7 GB)
- On Apple Silicon the image runs under x86-64 emulation (Rosetta/QEMU);
  startup takes ~2 minutes on first run, ~30 seconds on subsequent runs

### Quick start

```bash
# Start Splunk and run all integration tests (Splunk stays running afterwards)
make integration-test

# Tear down when done
make splunk-down
```

### Step-by-step

```bash
# 1. Start Splunk (exports SPLUNK_HOST and SPLUNK_TOKEN to the current shell)
eval "$(scripts/splunk-up.sh)"

# 2. Run integration tests
go test -v -tags integration -timeout 5m ./internal/client/...

# 3. Run a specific test
go test -v -tags integration -run TestIntegration_SearchAndResults ./internal/client/...

# 4. Tear down
scripts/splunk-down.sh
```

### Environment variables

| Variable | Description |
|----------|-------------|
| `SPLUNK_HOST` | Base URL of the Splunk REST API, e.g. `https://localhost:18503` |
| `SPLUNK_TOKEN` | Session token obtained via `splunk-up.sh` |

The integration test file (`internal/client/client_integration_test.go`) skips
all tests if either variable is unset, so running `make test` never accidentally
hits a real Splunk instance.

### Container defaults

| Setting | Value |
|---------|-------|
| Image | `docker.io/splunk/splunk:9.4` |
| Container name | `splunk-test` |
| Admin password | `Admin1234!` |
| REST API port | random in 18000–18999 (avoids conflicts) |
| TLS | self-signed (tests use `--insecure`) |

## Release

See the [Release Process](CHANGELOG.md) and [cli-series conventions](../CONVENTIONS.md).

```bash
make check            # must pass before tagging
git tag vX.Y.Z
git push origin vX.Y.Z
make build-all        # produces dist/
# zip each binary and upload to GitHub Release
```
