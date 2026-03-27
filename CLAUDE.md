# splunk-cli — CLAUDE.md

Project-specific instructions for Claude Code.
Series and org conventions: see `../CONVENTIONS.md` and
`https://github.com/nlink-jp/.github/blob/main/CONVENTIONS.md`.

## Architecture

```
cmd/splunk-cli/main.go    Entry point — injects version, calls cmd.Execute()
cmd/
  root.go                 Cobra root command, persistent flags, config loading
  common.go               Shared helpers (getSPL)
  run.go                  run command — synchronous search + Ctrl+C handling
  start.go                start command — async job, prints SID
  status.go               status command — prints SID/isDone/dispatchState
  results.go              results command — fetches results for a completed job
internal/
  config/
    config.go             Config struct, Load(), ApplyEnvVars(), permission check
    config_test.go
  client/
    client.go             Splunk REST API client
    client_test.go
config.example.toml       Template config file
```

## Config format

TOML with a `[splunk]` section. Default path: `~/.config/splunk-cli/config.toml`.

Priority: CLI flags > env vars (`SPLUNK_HOST`, `SPLUNK_TOKEN`, etc.) > config file.

## Key decisions

- **Cobra** for CLI (series standard); all connection flags are persistent (root-level).
- **Config permission check** — warns if file is group/world-readable (org security policy).
- **HTTP warning** — warns when sending a token over HTTP (org security policy).
- **`context.Context` everywhere** — all client methods accept ctx for cancellation/timeout.
- **`run` Ctrl+C** — prompts cancel/detach; second Ctrl+C cancels the job unconditionally.

## Testing

```
make test    # go test ./...
make check   # vet + lint + test + build
```
