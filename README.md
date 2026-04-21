# Luno MCP Server

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=luno_luno-mcp&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=luno_luno-mcp)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=luno_luno-mcp&metric=coverage)](https://sonarcloud.io/summary/new_code?id=luno_luno-mcp)
[![Bugs](https://sonarcloud.io/api/project_badges/measure?project=luno_luno-mcp&metric=bugs)](https://sonarcloud.io/summary/new_code?id=luno_luno-mcp)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=luno_luno-mcp&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=luno_luno-mcp)
[![Vulnerabilities](https://sonarcloud.io/api/project_badges/measure?project=luno_luno-mcp&metric=vulnerabilities)](https://sonarcloud.io/summary/new_code?id=luno_luno-mcp)
[![Duplicated Lines (%)](https://sonarcloud.io/api/project_badges/measure?project=luno_luno-mcp&metric=duplicated_lines_density)](https://sonarcloud.io/summary/new_code?id=luno_luno-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/luno/luno-mcp)](https://goreportcard.com/report/github.com/luno/luno-mcp)
[![GoDoc](https://godoc.org/github.com/luno/luno-mcp?status.svg)](https://godoc.org/github.com/luno/luno-mcp)

A [Model Context Protocol](https://modelcontextprotocol.io) (MCP) server that provides access to the Luno cryptocurrency exchange API.

This server enables integration with Claude Code/VSCode/Cursor (and other MCP-compatible clients), providing contextual information and functionality related to the Luno cryptocurrency exchange.

## Getting started

Some tools require your Luno API key and secret. Get these from your [Luno account settings](https://www.luno.com/developers).

[<img src="https://img.shields.io/badge/VS_Code-VS_Code?style=flat-square&label=Install%20Server&color=0098FF" alt="Install in VS Code">](https://insiders.vscode.dev/redirect?url=vscode%3Amcp%2Finstall%3F%257B%2522name%2522%253A%2522luno-mcp%2522%252C%2522command%2522%253A%2522docker%2522%252C%2522args%2522%253A%255B%2522run%2522%252C%2522--rm%2522%252C%2522-i%2522%252C%2522-e%2522%252C%2522LUNO_API_KEY_ID%253D%2524%257Binput%253Aluno_api_key_id%257D%2522%252C%2522-e%2522%252C%2522LUNO_API_SECRET%253D%2524%257Binput%253Aluno_api_secret%257D%2522%252C%2522ghcr.io%252Fluno%252Fluno-mcp%253Alatest%2522%255D%252C%2522inputs%2522%253A%255B%257B%2522id%2522%253A%2522luno_api_key_id%2522%252C%2522type%2522%253A%2522promptString%2522%252C%2522description%2522%253A%2522Luno%2520API%2520Key%2520ID%2522%252C%2522password%2522%253Atrue%257D%252C%257B%2522id%2522%253A%2522luno_api_secret%2522%252C%2522type%2522%253A%2522promptString%2522%252C%2522description%2522%253A%2522Luno%2520API%2520Secret%2522%252C%2522password%2522%253Atrue%257D%255D%257D) [<img alt="Install in VS Code Insiders" src="https://img.shields.io/badge/VS_Code_Insiders-VS_Code_Insiders?style=flat-square&label=Install%20Server&color=24bfa5">](https://insiders.vscode.dev/redirect?url=vscode-insiders%3Amcp%2Finstall%3F%257B%2522name%2522%253A%2522luno-mcp%2522%252C%2522command%2522%253A%2522docker%2522%252C%2522args%2522%253A%255B%2522run%2522%252C%2522--rm%2522%252C%2522-i%2522%252C%2522-e%2522%252C%2522LUNO_API_KEY_ID%253D%2524%257Binput%253Aluno_api_key_id%257D%2522%252C%2522-e%2522%252C%2522LUNO_API_SECRET%253D%2524%257Binput%253Aluno_api_secret%257D%2522%252C%2522ghcr.io%252Fluno%252Fluno-mcp%253Alatest%2522%255D%252C%2522inputs%2522%253A%255B%257B%2522id%2522%253A%2522luno_api_key_id%2522%252C%2522type%2522%253A%2522promptString%2522%252C%2522description%2522%253A%2522Luno%2520API%2520Key%2520ID%2522%252C%2522password%2522%253Atrue%257D%252C%257B%2522id%2522%253A%2522luno_api_secret%2522%252C%2522type%2522%253A%2522promptString%2522%252C%2522description%2522%253A%2522Luno%2520API%2520Secret%2522%252C%2522password%2522%253Atrue%257D%255D%257D) [<img src="https://cursor.com/deeplink/mcp-install-dark.svg" alt="Install in Cursor">](https://cursor.com/en/install-mcp?name=luno-mcp&config=eyJjb21tYW5kIjoiZG9ja2VyIiwiYXJncyI6WyJydW4iLCItLXJtIiwiLWkiLCItZSIsIkxVTk9fQVBJX0tFWV9JRD1ZT1VSX0FQSV9LRVlfSUQiLCItZSIsIkxVTk9fQVBJX1NFQ1JFVD1ZT1VSX0FQSV9TRUNSRVQiLCJnaGNyLmlvL2x1bm8vbHVuby1tY3A6bGF0ZXN0Il19)

**Standard config** (Docker) works with most MCP clients:

```json
{
  "mcpServers": {
    "luno": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LUNO_API_KEY_ID=YOUR_API_KEY_ID",
        "-e", "LUNO_API_SECRET=YOUR_API_SECRET",
        "ghcr.io/luno/luno-mcp:latest"
      ]
    }
  }
}
```

<details>
<summary>Claude Code</summary>

Using Docker:

```bash
claude mcp add luno -- docker run --rm -i -e LUNO_API_KEY_ID=YOUR_API_KEY_ID -e LUNO_API_SECRET=YOUR_API_SECRET ghcr.io/luno/luno-mcp:latest
```

Or if you've built from source:

```bash
claude mcp add luno -e LUNO_API_KEY_ID=YOUR_API_KEY_ID -e LUNO_API_SECRET=YOUR_API_SECRET -- luno-mcp
```

</details>

<details>
<summary>Claude Desktop</summary>

One-line install and configure (macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/luno/luno-mcp/main/claude-desktop-install.sh | \
  LUNO_API_KEY_ID=<key> LUNO_API_SECRET=<secret> sh
```

Or add the standard config to your `claude_desktop_config.json` manually ([setup guide](https://modelcontextprotocol.io/quickstart/user)).

</details>

<details>
<summary>Cursor</summary>

Go to `Cursor Settings` → `MCP` → `Add new MCP Server`. Use `command` type with command `docker` and the args from the standard config above.

Or add the standard config to your `.cursor/mcp.json`.

</details>

<details>
<summary>Windsurf</summary>

Follow the Windsurf MCP [documentation](https://docs.windsurf.com/windsurf/cascade/mcp). Use the standard config above.

</details>

<details>
<summary>VS Code (manual configuration)</summary>

Click the badges above for one-click Docker install, or add the following to your VS Code `settings.json` or `.vscode/mcp.json`:

### With Docker

```json
{
  "inputs": [
    {"id": "luno_api_key_id", "type": "promptString", "description": "Luno API Key ID", "password": true},
    {"id": "luno_api_secret", "type": "promptString", "description": "Luno API Secret", "password": true}
  ],
  "servers": {
    "luno": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LUNO_API_KEY_ID=${input:luno_api_key_id}",
        "-e", "LUNO_API_SECRET=${input:luno_api_secret}",
        "ghcr.io/luno/luno-mcp:latest"
      ]
    }
  }
}
```

### From source

```json
{
  "inputs": [
    {"id": "luno_api_key_id", "type": "promptString", "description": "Luno API Key ID", "password": true},
    {"id": "luno_api_secret", "type": "promptString", "description": "Luno API Secret", "password": true}
  ],
  "servers": {
    "luno": {
      "command": "luno-mcp",
      "env": {
        "LUNO_API_KEY_ID": "${input:luno_api_key_id}",
        "LUNO_API_SECRET": "${input:luno_api_secret}"
      }
    }
  }
}
```

### SSE transport

```json
{
  "servers": {
    "luno": {
      "type": "sse",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

</details>

<details>
<summary>Building from source</summary>

Requires Go 1.25 or later.

Install directly:

```bash
go install github.com/luno/luno-mcp/cmd/server@latest
```

Or clone and build:

```bash
git clone https://github.com/luno/luno-mcp
cd luno-mcp
go build -o luno-mcp ./cmd/server
```

Optionally make it available system-wide:

```bash
sudo mv luno-mcp /usr/local/bin/
```

</details>

<details>
<summary>Homebrew (macOS)</summary>

Install via [Homebrew](https://brew.sh) using the [luno/homebrew-luno-mcp](https://github.com/luno/homebrew-luno-mcp) tap:

```bash
brew tap luno/luno-mcp
brew install luno-mcp
```

Then configure your MCP client:

```json
{
  "mcpServers": {
    "luno": {
      "command": "luno-mcp",
      "args": ["--transport", "stdio"],
      "env": {
        "LUNO_API_KEY_ID": "YOUR_API_KEY_ID",
        "LUNO_API_SECRET": "YOUR_API_SECRET"
      }
    }
  }
}
```

Or with Claude Code:

```bash
claude mcp add luno -e LUNO_API_KEY_ID=YOUR_API_KEY_ID -e LUNO_API_SECRET=YOUR_API_SECRET -- luno-mcp
```

</details>

<details>
<summary>Docker</summary>

Use the standard config above, or run directly:

```bash
docker run --rm -i \
  -e LUNO_API_KEY_ID=YOUR_API_KEY_ID \
  -e LUNO_API_SECRET=YOUR_API_SECRET \
  ghcr.io/luno/luno-mcp:latest
```

For SSE mode:

```bash
docker run --rm \
  -e LUNO_API_KEY_ID=YOUR_API_KEY_ID \
  -e LUNO_API_SECRET=YOUR_API_SECRET \
  -p 8080:8080 \
  ghcr.io/luno/luno-mcp:latest \
  --transport sse --sse-address 0.0.0.0:8080
```

Optional environment variables:
- `LUNO_API_DEBUG=true` — Enable debug logging
- `LUNO_API_DOMAIN=api.staging.luno.com` — Override API domain
- `ALLOW_WRITE_OPERATIONS=true` — Enable write operations (`create_order`, `cancel_order`)

</details>

**Standard config** (Docker) works with most MCP clients:

```json
{
  "mcpServers": {
    "luno": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LUNO_API_KEY_ID=YOUR_API_KEY_ID",
        "-e", "LUNO_API_SECRET=YOUR_API_SECRET",
        "ghcr.io/luno/luno-mcp:latest"
      ]
    }
  }
}
```

<details>
<summary>Claude Code</summary>

Using Docker:

```bash
claude mcp add luno -- docker run --rm -i -e LUNO_API_KEY_ID=YOUR_API_KEY_ID -e LUNO_API_SECRET=YOUR_API_SECRET ghcr.io/luno/luno-mcp:latest
```

Or if you've built from source:

```bash
claude mcp add luno -e LUNO_API_KEY_ID=YOUR_API_KEY_ID -e LUNO_API_SECRET=YOUR_API_SECRET -- luno-mcp
```

</details>

<details>
<summary>Claude Desktop</summary>

One-line install and configure (macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/luno/luno-mcp/main/claude-desktop-install.sh | \
  LUNO_API_KEY_ID=<key> LUNO_API_SECRET=<secret> sh
```

Or add the standard config to your `claude_desktop_config.json` manually ([setup guide](https://modelcontextprotocol.io/quickstart/user)).

</details>

<details>
<summary>Cursor</summary>

Go to `Cursor Settings` → `MCP` → `Add new MCP Server`. Use `command` type with command `docker` and the args from the standard config above.

Or add the standard config to your `.cursor/mcp.json`.

</details>

<details>
<summary>Windsurf</summary>

Follow the Windsurf MCP [documentation](https://docs.windsurf.com/windsurf/cascade/mcp). Use the standard config above.

</details>

<details>
<summary>VS Code (manual configuration)</summary>

Click the badge above for one-click Docker install, or add the following to your VS Code `settings.json` or `.vscode/mcp.json`:

#### With Docker

```json
{
  "servers": {
    "luno": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LUNO_API_KEY_ID=${input:luno_api_key_id}",
        "-e", "LUNO_API_SECRET=${input:luno_api_secret}",
        "ghcr.io/luno/luno-mcp:latest"
      ],
      "inputs": [
         {"id": "luno_api_key_id", "type": "promptString", "description": "Luno API Key ID", "password": true},
         {"id": "luno_api_secret", "type": "promptString", "description": "Luno API Secret", "password": true}
      ]
    }
  }
}
```

#### From source

```json
{
  "servers": {
    "luno": {
      "command": "luno-mcp",
      "env": {
        "LUNO_API_KEY_ID": "${input:luno_api_key_id}",
        "LUNO_API_SECRET": "${input:luno_api_secret}"
      },
      "inputs": [
        {"id": "luno_api_key_id", "type": "promptString", "description": "Luno API Key ID", "password": true},
        {"id": "luno_api_secret", "type": "promptString", "description": "Luno API Secret", "password": true}
      ]
    }
  }
}
```

#### SSE transport

```json
{
  "servers": {
    "luno": {
      "type": "sse",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

</details>

<details>
<summary>Building from source</summary>

Requires Go 1.25 or later.

Install directly:

```bash
go install github.com/luno/luno-mcp/cmd/server@latest
```

Or clone and build:

```bash
git clone https://github.com/luno/luno-mcp
cd luno-mcp
go build -o luno-mcp ./cmd/server
```

Optionally make it available system-wide:

```bash
sudo mv luno-mcp /usr/local/bin/
```

</details>

<details>
<summary>Docker</summary>

Use the standard config above, or run directly:

```bash
docker run --rm -i \
  -e LUNO_API_KEY_ID=YOUR_API_KEY_ID \
  -e LUNO_API_SECRET=YOUR_API_SECRET \
  ghcr.io/luno/luno-mcp:latest
```

For SSE mode:

```bash
docker run --rm \
  -e LUNO_API_KEY_ID=YOUR_API_KEY_ID \
  -e LUNO_API_SECRET=YOUR_API_SECRET \
  -p 8080:8080 \
  ghcr.io/luno/luno-mcp:latest \
  --transport sse --sse-address 0.0.0.0:8080
```

Optional environment variables:
- `LUNO_API_DEBUG=true` — Enable debug logging
- `LUNO_API_DOMAIN=api.staging.luno.com` — Override API domain
- `ALLOW_WRITE_OPERATIONS=true` — Enable write operations (`create_order`, `cancel_order`)

</details>

## Features

- **Resources**: Access to account balances, transaction history, and more
- **Tools**: Functionality for creating and managing orders, checking prices, and viewing transaction details
- **Security**: Secure authentication using Luno API keys
- **VS Code Integration**: Easy integration with VSCode, or other AI IDEs

## Available Tools

| Tool                | Category            | Description                                       | Auth Required | Write |
| ------------------- | ------------------- | ------------------------------------------------- | ------------- | ----- |
| `get_ticker`        | Market Data         | Get current ticker information for a trading pair | ❌            | ❌    |
| `get_tickers`       | Market Data         | List tickers for given pairs (or all)             | ❌            | ❌    |
| `get_order_book`    | Market Data         | Get the order book for a trading pair             | ❌            | ❌    |
| `list_trades`       | Market Data         | List recent trades for a currency pair            | ❌            | ❌    |
| `get_candles`       | Market Data         | Get candlestick market data for a currency pair   | ❌            | ❌    |
| `get_markets_info`  | Market Data         | List all supported markets parameter information  | ❌            | ❌    |
| `get_balances`      | Account Information | Get balances for all accounts                     | ✅            | ❌    |
| `create_order`      | Trading             | Create a new buy or sell order                    | ✅            | ✅    |
| `cancel_order`      | Trading             | Cancel an existing order                          | ✅            | ✅    |
| `list_orders`       | Trading             | List open orders                                  | ✅            | ❌    |
| `list_transactions` | Transactions        | List transactions for an account                  | ✅            | ❌    |
| `get_transaction`   | Transactions        | Get details of a specific transaction             | ✅            | ❌    |

## Command-line options

- `--transport`: Transport type (`stdio`, `sse`, or `streamable-http`; default: `streamable-http`)
- `--sse-address`: Address for SSE and Streamable HTTP transports (default: `localhost:8080`)
- `--domain`: Luno API domain (default: `api.luno.com`)
- `--log-level`: Log level (`debug`, `info`, `warn`, `error`, default: `info`)
- `--allow-write-operations`: Enable write operations (`create_order`, `cancel_order`). Also configurable via `ALLOW_WRITE_OPERATIONS` env var

## Examples

### Working with wallets

You can ask your LLM to show your wallet balances:

```text
What are my current wallet balances on Luno?
```

### Trading

You can ask your LLM to help you trade:

```text
Create a limit order to buy 0.001 BTC at 50000 ZAR
```

### Transaction history

You can ask your LLM to show your transaction history:

```text
Show me my recent Bitcoin transactions
```

### Market Data

You can ask your LLM to show market data:

```text
Show me recent trades for XBTZAR
```

```text
What's the latest price for Bitcoin in ZAR?
```

## Security Considerations

This tool requires API credentials that have access to your Luno account. Be cautious when using API keys, especially ones with withdrawal permissions. It's recommended to create API keys with only the permissions needed for your specific use case.

### Write Operations Control

By default, the MCP server runs in **read-only mode** — `create_order` and `cancel_order` are not exposed. To enable them, set `ALLOW_WRITE_OPERATIONS` to `true`, `1`, or `yes`. See the config examples above for where to add this flag.

### Best Practices for API Credentials

1. **Create Limited-Permission API Keys**: Only grant the permissions absolutely necessary for your use case
2. **Never Commit Credentials to Version Control**: Ensure `.env` files are always in your `.gitignore`
3. **Rotate API Keys Regularly**: Periodically regenerate your API keys to limit the impact of potential leaks
4. **Monitor API Usage**: Regularly check your Luno account for any unauthorized activity
5. **Use Read-Only Mode by Default**: Only enable write operations when specifically needed

## Contributing

If you'd like to contribute to the development of this project, please see the [CONTRIBUTING.md](CONTRIBUTING.md) file for guidelines.

## License

[MIT License](LICENSE)
