# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Testing
- `make build` - Build the luno-mcp binary
- `make test` - Run all tests
- `make clean` - Clean build files and remove binary
- `make install` - Install binary to GOBIN path
- `make pre-commit` - Install pre-commit hooks

### Running the Server
- `make run-stdio` - Run server in stdio mode (default MCP transport)
- `make run-sse` - Run server in SSE mode on localhost:8080
- `go run ./cmd/server --transport sse --sse-address localhost:8080` - Custom SSE configuration

### Pre-commit Hooks
Pre-commit hooks are required and run automatically on commit:
- Go formatting (gofumpt, goimports)
- Go vet checks
- Go mod tidy
- YAML validation
- Trailing whitespace removal
- End-of-file fixing

Install with: `pre-commit install`

## Project Architecture

This is a Model Context Protocol (MCP) server providing access to the Luno cryptocurrency exchange API. The architecture follows Go standard project layout:

### Core Structure
- `cmd/server/` - Main application entry point with CLI flags and server startup
- `internal/config/` - Configuration handling (environment variables, API credentials)
- `internal/server/` - MCP server setup, transport handling (stdio/SSE)
- `internal/tools/` - MCP tools implementation for Luno API interactions
- `internal/resources/` - MCP resources for data exposure
- `internal/logging/` - Enhanced logging with MCP notification support
- `internal/tests/` - Testing utilities

### Key Dependencies
- `github.com/mark3labs/mcp-go` - MCP protocol implementation
- `github.com/luno/luno-go` - Official Luno API client
- `github.com/joho/godotenv` - Environment file loading
- `github.com/vektra/mockery/v3` - Mock generation (declared as Go tool)

## Configuration and Environment

### API Credentials Setup
For development, set credentials via:

**Environment variables:**
```bash
export LUNO_API_KEY_ID=your_key
export LUNO_API_SECRET=your_secret
export LUNO_API_DEBUG=true  # Optional
export LUNO_API_DOMAIN=api.staging.luno.com  # Optional
```

**Or `.env` file (gitignored):**
Copy `.env.example` to `.env` and populate values.

**Note:** MCP clients (VS Code, etc.) provide credentials through input prompts, not environment variables.

### Available Command-Line Flags
- `--transport` - Transport type (stdio or sse, default: stdio)
- `--sse-address` - SSE server address (default: localhost:8080)
- `--domain` - Luno API domain override
- `--log-level` - Logging level (debug, info, warn, error, default: info)

## Code Conventions

### Go Standards (from copilot-instructions.md)
- Follow Go idioms and best practices
- Always handle errors properly - either return them or log them, never both
- Rarely ignore errors; if you do, explicitly use `_ =`
- Use simple, readable code over clever solutions
- Don't worry about cyclomatic complexity in `*_test.go` files

### Testing Requirements
- Write table-driven tests with descriptive names using spaces (not underscores)
- Test happy path, boundary cases, and error conditions
- Use testify assertions or standard library
- Use `.EXPECT()` rather than `.On()` for mock expectations
- Run mockery as `go tool mockery` (not standalone `mockery`)

### MCP-Specific Guidelines
- Separate concerns between resources (data exposure) and tools (actions)
- Implement proper input validation for client requests
- Follow mcp-go library patterns
- Provide helpful error messages to users
- Handle API rate limiting and failures gracefully

### Commit Message Format
Follow pattern: `"<package>/<optional_subpackage>: <Capital description>"` in present tense.

Examples:
- `"config: Trim spaces when parsing env vars"`
- `"logging: Use mcp logging package so it's not interpreted as errors"`
- `"tools: Add validation for trading pairs before order creation"`

### Security Best Practices
- Never store or log API credentials
- Validate all inputs, especially from external sources
- Don't log sensitive information (keys, full API responses)
- Only log information needed for debugging
- Consider edge cases and failure modes

## Docker Support

Build: `docker build -t luno-mcp .`

Run with environment file: `docker run --env-file .env luno-mcp`

SSE mode: `docker run --env-file .env -p 8080:8080 luno-mcp --transport sse --sse-address 0.0.0.0:8080`
