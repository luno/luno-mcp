package server

import (
	"context"
	"log/slog"
	"os"

	"github.com/luno/luno-mcp/internal/config"
	"github.com/luno/luno-mcp/internal/resources"
	"github.com/luno/luno-mcp/internal/tools"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// NewMCPServer creates a new MCP server
func NewMCPServer(name, version string, cfg *config.Config, hooks ...*mcpserver.Hooks) *mcpserver.MCPServer {
	// Prepare options for the server
	options := []mcpserver.ServerOption{
		mcpserver.WithResourceCapabilities(true, true),
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithLogging(),
	}

	// Add hooks if provided
	for _, hook := range hooks {
		options = append(options, mcpserver.WithHooks(hook))
	}

	// Create server with capabilities
	server := mcpserver.NewMCPServer(
		name,
		version,
		options...,
	)

	// Register resources
	registerResources(server, cfg)

	// Register tools
	registerTools(server, cfg)

	return server
}

// registerResources registers all resources with the MCP server
func registerResources(server *mcpserver.MCPServer, cfg *config.Config) {
	// Add balance resources
	walletResource := resources.NewWalletResource()
	server.AddResource(walletResource, resources.HandleWalletResource(cfg))

	// Add transactions resource
	transactionsResource := resources.NewTransactionsResource()
	server.AddResource(transactionsResource, resources.HandleTransactionsResource(cfg))

	// Add account resource template
	accountTemplate := resources.NewAccountTemplate()
	server.AddResourceTemplate(accountTemplate, resources.HandleAccountTemplate(cfg))
}

// registerTools registers all tools with the MCP server
func registerTools(server *mcpserver.MCPServer, cfg *config.Config) {
	// Add balance tools
	balancesTool := tools.NewGetBalancesTool()
	server.AddTool(balancesTool, tools.HandleGetBalances(cfg))

	// Add market tools
	tickerTool := tools.NewGetTickerTool()
	server.AddTool(tickerTool, tools.HandleGetTicker(cfg))

	orderBookTool := tools.NewGetOrderBookTool()
	server.AddTool(orderBookTool, tools.HandleGetOrderBook(cfg))

	// Add trading tools
	// Only add write operation tools if explicitly allowed
	if cfg.AllowWriteOperations {
		slog.Info("Write operations enabled - registering create_order and cancel_order tools")
		createOrderTool := tools.NewCreateOrderTool()
		server.AddTool(createOrderTool, tools.HandleCreateOrder(cfg))

		cancelOrderTool := tools.NewCancelOrderTool()
		server.AddTool(cancelOrderTool, tools.HandleCancelOrder(cfg))
	} else {
		slog.Info("Write operations disabled - create_order and cancel_order tools will not be available")
	}

	listOrdersTool := tools.NewListOrdersTool()
	server.AddTool(listOrdersTool, tools.HandleListOrders(cfg))

	// Add transaction tools
	listTransactionsTool := tools.NewListTransactionsTool()
	server.AddTool(listTransactionsTool, tools.HandleListTransactions(cfg))

	getTransactionTool := tools.NewGetTransactionTool()
	server.AddTool(getTransactionTool, tools.HandleGetTransaction(cfg))

	// Add trades tools
	listTradesTool := tools.NewListTradesTool()
	server.AddTool(listTradesTool, tools.HandleListTrades(cfg))
}

// ServeStdio starts the server using the Stdio transport
func ServeStdio(ctx context.Context, s *mcpserver.MCPServer) error {
	stdioServer := mcpserver.NewStdioServer(s)

	// Create context function that adds authentication
	contextFunc := func(ctx context.Context) context.Context {
		return ctx
	}

	stdioServer.SetContextFunc(contextFunc)

	// Listen on stdin/stdout
	return stdioServer.Listen(ctx, os.Stdin, os.Stdout)
}

// ServeSSE starts the server using the SSE transport
func ServeSSE(ctx context.Context, s *mcpserver.MCPServer, addr string) error {
	sseServer := mcpserver.NewSSEServer(s)

	// Start the server
	slog.Info("SSE server listening on " + addr)
	return sseServer.Start(addr)
}
