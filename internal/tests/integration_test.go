package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/luno/luno-go"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/luno/luno-mcp/internal/server"
	"github.com/luno/luno-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestMainWithListOrdersIntegration simulates running main with list_orders
func TestMainWithListOrdersIntegration(t *testing.T) {
	t.Skip("integration test")
	// Setup: Load environment and create config
	cfg, err := setupTestConfig(t)
	if err != nil {
		t.Fatalf("Failed to set up config: %v", err)
	}

	// Skip creating an MCP server, instead directly call the Luno client
	// to test the functionality that would be exposed through the list_orders tool

	// Create a context
	ctx := context.Background()

	// Create the list orders request similar to what the tool would use
	listReq := &luno.ListOrdersRequest{
		// No parameters to keep it simple, just test the API connection
		Limit: 100, // Default value
	}

	// Call the Luno API
	orders, err := cfg.LunoClient.ListOrders(ctx, listReq)
	if err != nil {
		t.Fatalf("Failed to list orders: %v", err)
	}

	// Verify the response
	if orders == nil {
		t.Fatal("Orders response is nil")
	}

	// Print order count
	t.Logf("Successfully retrieved %d orders", len(orders.Orders))

	// Pretty print orders for inspection
	ordersJSON, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal orders to JSON: %v", err)
	}
	t.Logf("Orders: %s", string(ordersJSON))
}

// TestMCPServerWithListOrders tests the MCP server setup with list_orders tool
func TestMCPServerWithListOrders(t *testing.T) {
	t.Skip("integration test")
	// Setup: Load environment and create config
	cfg, err := setupTestConfig(t)
	if err != nil {
		t.Fatalf("Failed to set up config: %v", err)
	}

	// Create MCP server and register tools
	mcpServer := server.NewMCPServer("luno-mcp-test", "0.1.0", cfg)

	// Verify the server was created successfully
	if mcpServer == nil {
		t.Fatal("Failed to create MCP server")
	}

	// Get the list_orders tool handler
	listOrdersHandler := tools.HandleListOrders(cfg)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a simple request - just enough to satisfy the handler
	request := mcp.CallToolRequest{}
	request.Method = "callTool"
	request.Params.Name = tools.ListOrdersToolID
	request.Params.Arguments = make(map[string]interface{})

	// Call the tool handler directly
	result, err := listOrdersHandler(ctx, request)
	if err != nil {
		t.Fatalf("Failed to call list_orders handler: %v", err)
	}

	// Verify we got a valid result
	if result == nil {
		t.Fatal("List orders handler returned nil result")
	}

	// Print the result information
	t.Logf("List orders handler returned result with %d content items", len(result.Content))

	// Check if there is any content
	if len(result.Content) > 0 {
		// Success!
		t.Log("List orders tool returned content successfully")
	} else if result.IsError {
		t.Fatalf("List orders tool returned an error")
	}
}

// setupTestConfig loads environment variables and creates a config
func setupTestConfig(t *testing.T) (*config.Config, error) {
	// Try different possible locations for the .env file
	envPaths := []string{
		"../../.env", // Relative to tests directory
		"../.env",    // One level up
		".env",       // Current directory
	}

	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			t.Logf("Successfully loaded environment from %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		t.Log("Warning: No .env file found, using environment variables from system")
	}

	return config.Load("", "test-app", "1.0.0")
}
