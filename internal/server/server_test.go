package server

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/luno/luno-go"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/luno/luno-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

const (
	testServerName       = "test-server"
	testServerWithHooks  = "test-server-with-hooks"
	testServerMultiHooks = "test-server-multi-hooks"
	testVersion1         = "1.0.0"
	testVersion2         = "1.0.1"
	testVersion3 = "1.0.2"
)

func TestNewMCPServer(t *testing.T) {
	tests := []struct {
		name              string
		srvName           string
		version           string
		hooks             []*mcpserver.Hooks
		allowWriteOps     bool
		expectedToolCount int
	}{
		{
			name:              "creates server without hooks and write ops disabled",
			srvName:           testServerName,
			version:           testVersion1,
			hooks:             nil,
			allowWriteOps:     false,
			expectedToolCount: 12,
		},
		{
			name:              "creates server with write ops enabled",
			srvName:           testServerName,
			version:           testVersion1,
			hooks:             nil,
			allowWriteOps:     true,
			expectedToolCount: 12,
		},
		{
			name:              "creates server with single hook",
			srvName:           testServerWithHooks,
			version:           testVersion2,
			allowWriteOps:     false,
			expectedToolCount: 12,
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
			name:              "creates server with multiple distinct hook objects",
			srvName:           testServerMultiHooks,
			version:           testVersion3,
			allowWriteOps:     false,
			expectedToolCount: 12,
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
			cfg := &config.Config{
				LunoClient:           lunoClient,
				AllowWriteOperations: tc.allowWriteOps,
			}

			server := NewMCPServer(tc.srvName, tc.version, cfg, tc.hooks...)

			require.NotNil(t, server, "NewMCPServer should return non-nil server")
			require.Equal(t, tc.expectedToolCount, len(server.ListTools()), "unexpected number of registered tools")
		})
	}
}

func TestWriteOperationsControl(t *testing.T) {
	tests := []struct {
		name          string
		allowWriteOps bool
	}{
		{
			name:          "write operations disabled by default",
			allowWriteOps: false,
		},
		{
			name:          "write operations enabled when flag is true",
			allowWriteOps: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lunoClient := luno.NewClient()
			cfg := &config.Config{
				LunoClient:           lunoClient,
				AllowWriteOperations: tc.allowWriteOps,
			}

			srv := NewMCPServer("test-write-ops", "1.0.0", cfg)
			require.NotNil(t, srv, "NewMCPServer should return non-nil server")

			// Write operation tools should always be registered regardless of the flag
			registeredTools := srv.ListTools()
			require.Contains(t, registeredTools, tools.CreateOrderToolID,
				"%s: expected %s tool to always be registered", tc.name, tools.CreateOrderToolID)
			require.Contains(t, registeredTools, tools.CancelOrderToolID,
				"%s: expected %s tool to always be registered", tc.name, tools.CancelOrderToolID)

			// When disabled, verify the server routes calls to the disabled handler
			if !tc.allowWriteOps {
				for _, toolID := range []string{tools.CreateOrderToolID, tools.CancelOrderToolID} {
					resp := callTool(t, srv, toolID)
					require.Contains(t, resp, tools.ErrWriteOperationDisabled,
						"%s: calling %s should return disabled error", tc.name, toolID)
				}
			}
		})
	}
}

// callTool invokes a tool through the MCP server's HandleMessage entry point
// and returns the text content from the response.
func callTool(t *testing.T, srv *mcpserver.MCPServer, toolID string) string {
	t.Helper()

	msg := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":{}}}`, toolID)
	result := srv.HandleMessage(context.Background(), json.RawMessage(msg))

	b, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(b, &parsed))
	require.True(t, parsed.Result.IsError, "expected tool call to return an error result")
	require.NotEmpty(t, parsed.Result.Content, "expected at least one content item")
	return parsed.Result.Content[0].Text
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
			errorMsg: "unknown port",
		},
		{
			name:     "invalid port",
			address:  "localhost:99999",
			errorMsg: "invalid port",
		},
		{
			name:     "bind to used port",
			address:  "localhost:80", // Typically requires root privileges
			errorMsg: "permission denied",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a proper MCP server for testing
			lunoClient := luno.NewClient()
			cfg := &config.Config{
				LunoClient:           lunoClient,
				AllowWriteOperations: false,
			}
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
