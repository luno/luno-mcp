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

This server enables integration with VS Code's Copilot and other MCP-compatible clients, providing contextual information and functionality related to the Luno cryptocurrency exchange.

## ⚠️ Beta Warning

This project is currently in **beta phase**. While we've made every effort to ensure stability and reliability, you may encounter unexpected behavior or limitations. Please use it with care and consider the following:

- This MCP server config may change without prior notice
- Performance and reliability might not be optimal
- Not all Luno API endpoints are implemented yet

We welcome feedback and bug reports to help improve the project. Please report any issues you encounter via the [GitHub issue tracker](../../issues).

[![Install in VS Code with Docker](<https://img.shields.io/badge/VS_Code-Install_Luno_MCP_(Docker)-0098FF?style=flat-square&logo=visualstudiocode&logoColor=white>)](https://insiders.vscode.dev/redirect/mcp/install?name=luno-mcp&inputs=%5B%7B%22id%22%3A%22luno_api_key_id%22%2C%22type%22%3A%22promptString%22%2C%22description%22%3A%22Luno%20API%20Key%20ID%22%2C%22password%22%3Atrue%7D%2C%7B%22id%22%3A%22luno_api_secret%22%2C%22type%22%3A%22promptString%22%2C%22description%22%3A%22Luno%20API%20Secret%22%2C%22password%22%3Atrue%7D%5D&config=%7B%22command%22%3A%22docker%22%2C%22args%22%3A%5B%22run%22%2C%20%22--rm%22%2C%20%22-i%22%2C%20%22-e%22%2C%20%22LUNO_API_KEY_ID%3D%24%7Binput%3Aluno_api_key_id%7D%22%2C%20%22-e%22%2C%20%22LUNO_API_SECRET%3D%24%7Binput%3Aluno_api_secret%7D%22%2C%20%22ghcr.io%2Fluno%2Fluno-mcp%3Alatest%22%5D%7D)

## Features

- **Resources**: Access to account balances, transaction history, and more
- **Tools**: Functionality for creating and managing orders, checking prices, and viewing transaction details
- **Security**: Secure authentication using Luno API keys
- **VS Code Integration**: Easy integration with VS Code's Copilot features

## Usage

### Setting up credentials

The server requires your Luno API key and secret. These can be obtained from your Luno account settings, see here for more info: [https://www.luno.com/developers](https://www.luno.com/developers).

### Command-line options

- `--transport`: Transport type (`stdio` or `sse`, default: `stdio`)
- `--sse-address`: Address for SSE transport (default: `localhost:8080`)
- `--domain`: Luno API domain (default: `api.luno.com`)
- `--log-level`: Log level (`debug`, `info`, `warn`, `error`, default: `info`)

## Available Tools

| Tool                | Category            | Description                                       | Requires Write Operations |
| ------------------- | ------------------- | ------------------------------------------------- | ------------------------- |
| `get_ticker`        | Market Data         | Get current ticker information for a trading pair | No                        |
| `get_order_book`    | Market Data         | Get the order book for a trading pair             | No                        |
| `list_trades`       | Market Data         | List recent trades for a currency pair            | No                        |
| `get_balances`      | Account Information | Get balances for all accounts                     | No                        |
| `create_order`      | Trading             | Create a new buy or sell order                    | **Yes**                   |
| `cancel_order`      | Trading             | Cancel an existing order                          | **Yes**                   |
| `list_orders`       | Trading             | List open orders                                  | No                        |
| `list_transactions` | Transactions        | List transactions for an account                  | No                        |
| `get_transaction`   | Transactions        | Get details of a specific transaction             | No                        |

## Examples

### Working with wallets

You can ask Copilot to show your wallet balances:

```text
What are my current wallet balances on Luno?
```

### Trading

You can ask Copilot to help you trade:

```text
Create a limit order to buy 0.001 BTC at 50000 ZAR
```

### Transaction history

You can ask Copilot to show your transaction history:

```text
Show me my recent Bitcoin transactions
```

### Market Data

You can ask Copilot to show market data:

```text
Show me recent trades for XBTZAR
```

```text
What's the latest price for Bitcoin in ZAR?
```

## VS Code Integration

To integrate with VS Code, add the following to your settings.json file (or click on the badge at the top of this README for the docker config).

### With Docker

This configuration will make VS Code run the Docker container. Ensure Docker is running on your system.

```json
{
  "servers": {
    "luno-docker": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LUNO_API_KEY_ID=${input:luno_api_key_id}",
        "-e", "LUNO_API_SECRET=${input:luno_api_secret}",
        // Optional: Add debug info
        // "-e", "LUNO_API_DEBUG=true",
        // Optional: Override default API domain
        // "-e", "LUNO_API_DOMAIN=api.staging.luno.com",
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

### Building from source

**For MCP client usage**: Add one of the config options below to your VS Code `settings.json` or `mcp.json` file. The credentials will be provided through VS Code's input prompts.

**For direct development**: You'll also need to set up environment variables or a `.env` file as described in the [CONTRIBUTING.md](./CONTRIBUTING.md) file.

#### For stdio transport

```json
"mcp": {
  "servers": {
    "luno": {
      "command": "luno-mcp",
      "args": [],
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

#### For SSE transport

```json
"mcp": {
  "servers": {
    "luno": {
      "type": "sse",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

## Installation

### Prerequisites

- Go 1.24 or later
- Luno account with API key and secret

### Building from Source

1. Clone the repository:

   ```bash
   git clone https://github.com/luno/luno-mcp
   cd luno-mcp
   ```

2. Build the binary:

   ```bash
   go build -o luno-mcp ./cmd/server
   ```

3. Make it available system-wide (optional):

   ```bash
   sudo mv luno-mcp /usr/local/bin/
   ```

**Note**: When using with MCP clients like VS Code, credentials are provided through the client's input system. For direct development and testing, see the credential setup instructions in CONTRIBUTING.md.

## Security Considerations

This tool requires API credentials that have access to your Luno account. Be cautious when using API keys, especially ones with withdrawal permissions. It's recommended to create API keys with only the permissions needed for your specific use case.

### Write Operations Control

By default, the MCP server runs in **read-only mode** with write operations (`create_order` and `cancel_order`) disabled for security. To enable write operations, you must explicitly set the `ALLOW_WRITE_OPERATIONS` environment variable.

#### Enabling Write Operations

Set the environment variable to one of the following values:
- `ALLOW_WRITE_OPERATIONS=true`
- `ALLOW_WRITE_OPERATIONS=1`
- `ALLOW_WRITE_OPERATIONS=yes`

##### Docker Example:
```bash
docker run --rm -i \
  -e "LUNO_API_KEY_ID=${LUNO_API_KEY_ID}" \
  -e "LUNO_API_SECRET=${LUNO_API_SECRET}" \
  -e "ALLOW_WRITE_OPERATIONS=true" \
  ghcr.io/luno/luno-mcp:latest
```

##### VS Code Configuration:
```json
{
  "servers": {
    "luno-docker": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "LUNO_API_KEY_ID=${input:luno_api_key_id}",
        "-e", "LUNO_API_SECRET=${input:luno_api_secret}",
        "-e", "ALLOW_WRITE_OPERATIONS=true",
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

### Best Practices for API Credentials

1. **Create Limited-Permission API Keys**: Only grant the permissions absolutely necessary for your use case
2. **Never Commit Credentials to Version Control**: Ensure `.env` files are always in your `.gitignore`
3. **Rotate API Keys Regularly**: Periodically regenerate your API keys to limit the impact of potential leaks
4. **Monitor API Usage**: Regularly check your Luno account for any unauthorized activity
5. **Use Read-Only Mode by Default**: Only enable write operations when specifically needed

### Contributing

If you'd like to contribute to the development of this project, please see the [CONTRIBUTING.md](CONTRIBUTING.md) file for guidelines.

## License

[MIT License](LICENSE)
