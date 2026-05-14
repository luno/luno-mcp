# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development commands

- `make build` - Build the `luno-mcp` binary into the repo root
- `make test` - Run all tests (`go test ./...`)
- `go test ./internal/tools/... -run TestHandleGetBalances` - Run a single test or package
- `make run-stdio` / `make run-sse` / `make run-streamable-http` - Run server with each transport
- `make install` - Install binary to `$GOBIN`
- `make pre-commit` - Install pre-commit hooks (requires `gofumpt` installed separately: `go install mvdan.cc/gofumpt@latest`)
- `go tool mockery` - Regenerate mocks (mockery is declared as a Go tool in `go.mod`, not invoked standalone). Config is in `.mockery.yml`.

Pre-commit runs `go-vet-mod`, `go-imports`, `gofumpt`, `go-mod-tidy-repo`, `go-test-repo-mod`, plus YAML/whitespace/EOF checks and a script-test for `claude-desktop-install.sh`. These run on every commit - run `pre-commit run --all-files` to validate before pushing.

## Architecture

This is an MCP server wrapping the Luno crypto exchange API. Three layers matter:

**`sdk/luno_client.go`** defines the `LunoClient` interface that all tool/resource handlers depend on. The real `*luno.Client` satisfies it via a compile-time check; `sdk/mock_luno_client_gen.go` is the mockery-generated test double. **Never call `luno.Client` methods directly from `internal/` - go through the interface** so tests can substitute the mock.

**`internal/config`** owns `Config`, which carries the `LunoClient`, an `IsAuthenticated` flag, and `AllowWriteOperations`. `config.Load()` decides authentication mode by inspecting env vars at startup - if `LUNO_API_KEY_ID`/`LUNO_API_SECRET` are absent, the server runs in unauthenticated mode and authenticated tools (`get_balances`, `list_orders`, etc.) return `ErrAPICredentialsRequired` rather than failing at registration time. CLI `--allow-write-operations` overrides the env var.

**`internal/server/server.go`** wires everything: `NewMCPServer` registers resources and tools, then `ServeStdio` / `ServeSSE` / `ServeStreamableHTTP` start the chosen transport. Key design choice in `registerTools`: write tools (`create_order`, `cancel_order`) are **always registered with the server**, but when `AllowWriteOperations=false` they are bound to `HandleWriteOperationDisabled()` instead of their real handlers. This means MCP clients can see the tools exist and discover how to enable them, rather than the tools silently disappearing. Don't change this pattern when adding new write tools.

**Tool/resource split** follows MCP semantics: `internal/resources/` exposes data via URIs (`luno://wallets`, `luno://transactions`, `luno://accounts/{id}`); `internal/tools/` exposes actions (the table in README.md is the canonical list). Each tool file pairs a `NewXTool()` constructor (schema/description) with a `HandleX(cfg)` factory that closes over `*config.Config`.

**Transport-specific stdout handling**: in stdio mode, stdout is reserved for JSON-RPC frames, so `cmd/server/main.go:logWriter` routes slog output to stderr. Any new logging must go through `slog` (never `fmt.Println` to stdout) or it will corrupt the protocol stream.

**Dual logging via `internal/logging`**: `setupEnhancedLogger` wraps a console `slog.TextHandler` and an `MCPNotificationHandler` in a `MultiHandler`, so every `slog.Info`/`slog.Debug` call emits both a local log line *and* an MCP `notifications/message` to the client. The enhanced handler can only be installed after the MCP server exists, which is why the basic logger is set up first and replaced post-`createMCPServer`.

## Conventions

- Commit messages: `"<package>[/<subpackage>]: <Capital description>"` in present tense (e.g. `"config: Trim spaces when parsing env vars"`, `"tools: Add validation for trading pairs before order creation"`).
- Tests: table-driven, spaces (not underscores) in test names, `.EXPECT()` (not `.On()`) on mocks, testify or stdlib assertions. Cyclomatic complexity in `*_test.go` is not enforced.
- Error handling: return or log, never both. If ignoring an error, write `_ =` explicitly.
- Tool error responses: prefer `mcp.NewToolResultError(...)` with a user-actionable message (see the `ErrAPICredentialsRequired`, `ErrWriteOperationDisabled` constants in `internal/tools/tools.go` for the established phrasing) rather than returning a raw Go error.

## Configuration

API credentials and runtime options:

- `LUNO_API_KEY_ID`, `LUNO_API_SECRET` - credentials; absence triggers unauthenticated mode
- `LUNO_API_DOMAIN` - override default `api.luno.com` (e.g. `api.staging.luno.com`)
- `LUNO_API_DEBUG` - enable luno-go client debug mode
- `ALLOW_WRITE_OPERATIONS` - same effect as `--allow-write-operations` flag
- `.env` and `../.env` are auto-loaded at startup via godotenv (dev convenience only; MCP clients pass credentials via their own input mechanism)

CLI flags: `--transport` (stdio/sse/streamable-http, default `streamable-http`), `--sse-address` (default `localhost:8080`), `--domain`, `--log-level` (debug/info/warn/error), `--allow-write-operations`.

## Release

Releases are cut by publishing a GitHub Release on `main` with a `vX.Y.Z` tag. Two workflows fire in parallel: `release.yml` builds darwin/linux × arm64/amd64 binaries and dispatches an update event to [luno/homebrew-luno-mcp](https://github.com/luno/homebrew-luno-mcp); `docker-publish.yml` builds a multi-arch image to `ghcr.io/luno/luno-mcp` tagged with the version and `latest`. The `HOMEBREW_TAP_PAT` secret must be present for the Homebrew step.
