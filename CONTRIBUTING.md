# Contributing to Luno MCP Server

Thank you for your interest in contributing to the Luno MCP Server! This document provides guidelines and instructions to help you get started.

## Development Setup

### Prerequisites

- Go 1.24 or later
- [pre-commit](https://pre-commit.com/) for git hooks
- Luno account with API keys (for testing)

### Installing Pre-commit Hooks

We use pre-commit hooks to ensure code quality and consistency. The hooks will run automatically before each commit, checking for common issues and formatting your code.

1. Install pre-commit (if not already installed):

   ```bash
   # macOS
   brew install pre-commit

   # pip
   pip install pre-commit
   ```

2. Install the git hooks:

   ```bash
   pre-commit install
   ```

3. To manually run the hooks on all files:

   ```bash
   pre-commit run --all-files
   ```

### Setting up API Credentials for Development

For **direct development** (running the server outside of an MCP client), set credentials either through:

#### Environment Variables

Set the following environment variables in your shell:

```bash
export LUNO_API_KEY_ID=your_api_key_id
export LUNO_API_SECRET=your_api_secret
export LUNO_API_DEBUG=true                   # Optional: Enable debug mode
export LUNO_API_DOMAIN=staging.api.luno.com  # Optional: Override API Domain
```

#### An .env file

Copy the `.env.example` file and name it `.env` (this is always gitignored), and paste your keys in there:

```env
LUNO_API_KEY_ID=your_api_key_id
LUNO_API_SECRET=your_api_secret
LUNO_API_DEBUG=true                   # Optional: Enable debug mode
LUNO_API_DOMAIN=api.staging.luno.com  # Optional: Override API Domain
```

**Note**: When using the MCP server with VS Code or other MCP clients, credentials are provided through the client's input system. The `.env` file and environment variables are only needed for direct development when running `go run ./cmd/server` or the binary directly.

### Running the server

#### Standard I/O mode (default)

```bash
luno-mcp
```

#### Server-Sent Events (SSE) mode

```bash
luno-mcp --transport sse --sse-address localhost:8080
```

#### Using Docker

Build the Docker image:

```bash
docker build -t luno-mcp .
```

Run the Docker container with environment variables:

```bash
docker run \
   -e "LUNO_API_KEY_ID=${LUNO_API_KEY_ID}" \
   -e "LUNO_API_SECRET=${LUNO_API_SECRET}" \
   -e "LUNO_API_DOMAIN=api.staging.luno.com" \
   -e "LUNO_API_DEBUG=true" \
   luno-mcp
```

Alternatively, for convenience during development, you can use an `.env` file to provide these environment variables (this prevents your API key and secret from being stored in your shell history):

```bash
docker run --env-file .env luno-mcp
```

You can also use the `--transport sse` and `--sse-address` flags with Docker:

```bash
docker run --env-file .env -p 8080:8080 luno-mcp --transport sse --sse-address 0.0.0.0:8080
```

## Development Workflow

1. Create a new branch for your feature or bugfix (on a fork if you don't have permission to write to the repo):

   ```bash
   git checkout -b name-featureName
   ```

2. Make your changes and ensure they follow the project's coding conventions.

3. Run the tests to ensure your changes don't break existing functionality:

   ```bash
   make test
   ```

4. Build and test your changes locally:

   ```bash
   make build
   ./luno-mcp
   ```

5. Commit your changes. The pre-commit hooks will automatically run and may modify some files:

   ```bash
   git add .
   git commit -m "Your descriptive commit message"
   ```

6. Create a pull request.

## Code Style Guidelines

- Follow standard Go code conventions and idioms
- Use meaningful variable and function names
- Write clear comments for public APIs and complex logic
- Ensure error handling is comprehensive
- Keep functions focused and modular

## Release Process

Releases are fully automated and run on a daily schedule (9am UTC). A release is only created if there are new commits since the last tag — so merging to `main` is all that's needed. No manual tagging required.

The workflow can also be triggered manually via the [Actions tab](https://github.com/luno/luno-mcp/actions/workflows/release.yml) if you need to release outside the schedule.

### What happens automatically

#### Binaries + Homebrew (`release.yml`)

Runs daily at 9am UTC (or on manual trigger):

1. Checks for commits since the last release tag — skips if there are none
2. Bumps the patch version and creates a new tag (e.g. `v0.2.1` → `v0.2.2`)
3. Builds binaries for `darwin/arm64`, `darwin/amd64`, `linux/amd64`, and `linux/arm64`
4. Creates a GitHub release with the binaries as `.tar.gz` assets and `checksums.txt`
5. Dispatches a `new-release` event to the [luno/luno-mcp-homebrew](https://github.com/luno/luno-mcp-homebrew) tap repo, which auto-updates the formula with the new version and SHA256s

#### Docker image (`docker-publish.yml`)

Triggers on pushes to `main` and on `v*` tags. Builds and pushes a multi-arch image (`linux/amd64`, `linux/arm64`) to `ghcr.io/luno/luno-mcp`, tagged with the semver version and `latest`.

### Prerequisites for the release workflow

The `HOMEBREW_TAP_PAT` secret must be set in this repo's GitHub settings. It should be a fine-grained PAT with `Contents: write` permission on the `luno/luno-mcp-homebrew` repo.

### Installing via Homebrew (macOS)

Once a release is published, users can install with:

```bash
brew tap luno/luno-mcp-homebrew
brew install luno-mcp
```

## License

By contributing, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
