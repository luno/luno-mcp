package server

import (
	"context"
	"testing"

	"github.com/luno/luno-go"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp" // Added import
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

const (
	testServerName       = "test-server"
	testServerWithHooks  = "test-server-with-hooks"
	testServerMultiHooks = "test-server-multi-hooks"
	testVersion1         = "1.0.0"
	testVersion2         = "1.0.1"
	testVersion3         = "1.0.2"
	testVersion4         = "0.0.1"
)

func TestNewMCPServer(t *testing.T) {
	tests := []struct {
		name    string
		srvName string
		version string
		hooks   []*mcpserver.Hooks
	}{
		{
			name:    "creates server without hooks",
			srvName: testServerName,
			version: testVersion1,
			hooks:   nil,
		},
		{
			name:    "creates server with single hook",
			srvName: testServerWithHooks,
			version: testVersion2,
			hooks: []*mcpserver.Hooks{
				func() *mcpserver.Hooks {
					h := &mcpserver.Hooks{}
					h.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
						// Intentionally empty - testing hook registration, not hook execution.
					})
					return h
				}(),
			},
		},
		{
			name:    "creates server with multiple distinct hook objects",
			srvName: testServerMultiHooks,
			version: testVersion3,
			hooks: []*mcpserver.Hooks{
				func() *mcpserver.Hooks { // Corresponds to original OnAnyHookFunc
					h := &mcpserver.Hooks{}
					h.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
						// Intentionally empty - testing hook registration, not hook execution.
					})
					return h
				}(),
				func() *mcpserver.Hooks { // Corresponds to original BeforeAnyHookFunc
					h := &mcpserver.Hooks{}
					h.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
						// Intentionally empty - testing hook registration, not hook execution.
					})
					return h
				}(),
				func() *mcpserver.Hooks { // Corresponds to original AfterAnyHookFunc, using AddOnSuccess for generality
					h := &mcpserver.Hooks{}
					h.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
						// Intentionally empty - testing hook registration, not hook execution.
					})
					return h
				}(),
				func() *mcpserver.Hooks { // Corresponds to original OnErrorHookFunc
					h := &mcpserver.Hooks{}
					h.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
						// Intentionally empty - testing hook registration, not hook execution.
					})
					return h
				}(),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lunoClient := luno.NewClient()
			cfg := &config.Config{LunoClient: lunoClient}

			server := NewMCPServer(tc.srvName, tc.version, cfg, tc.hooks...)

			require.NotNil(t, server, "NewMCPServer should return non-nil server")

			// These should not panic
			registerResources(server, cfg)
			registerTools(server, cfg)
		})
	}
}

func TestServeSSEIntegration(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		errorMsg string
	}{
		{
			name:     "invalid address format",
			address:  "invalid:address",
			errorMsg: "lookup tcp/address: unknown port",
		},
		{
			name:     "invalid port",
			address:  "localhost:99999",
			errorMsg: "invalid port",
		},
		{
			name:     "bind to used port",
			address:  "localhost:80", // Typically requires root privileges
			errorMsg: "bind: permission denied",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a proper MCP server for testing
			lunoClient := luno.NewClient()
			cfg := &config.Config{LunoClient: lunoClient}
			server := NewMCPServer("test-sse-server", "1.0.0", cfg)

			// Set up context with or without timeout
			ctx := context.Background()
			// Test ServeSSE functionality
			err := ServeSSE(ctx, server, tc.address)

			if tc.errorMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServeStreamableHTTPIntegration(t *testing.T) {
	tests := []struct {
		name     string
		address  string
		errorMsg string
	}{
		{
			name:     "invalid address format",
			address:  "invalid:address",
			errorMsg: "lookup tcp/address: unknown port",
		},
		{
			name:     "invalid port",
			address:  "localhost:99999",
			errorMsg: "invalid port",
		},
		{
			name:     "bind to used port",
			address:  "localhost:80", // Typically requires root privileges
			errorMsg: "bind: permission denied",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a proper MCP server for testing
			lunoClient := luno.NewClient()
			cfg := &config.Config{LunoClient: lunoClient}
			server := NewMCPServer("test-streamable-http-server", "1.0.0", cfg)

			// Set up context
			ctx := context.Background()
			// Test ServeStreamableHTTP functionality
			err := ServeStreamableHTTP(ctx, server, tc.address)

			if tc.errorMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
