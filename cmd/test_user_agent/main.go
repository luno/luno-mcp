package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/luno/luno-go"
	"github.com/luno/luno-mcp/internal/config"
)

func main() {
	// Set up test environment variables
	os.Setenv("LUNO_API_KEY_ID", "test_key_id")
	os.Setenv("LUNO_API_SECRET", "test_secret")

	// Create a test server that captures and prints the User-Agent header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")
		fmt.Printf("Captured User-Agent: %s\n", userAgent)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Override the Luno domain to point to our test server
	serverURL := server.URL // Keep the full URL with http://
	
	// Load config which will create the Luno client with our MCP wrapper
	cfg, err := config.Load("", "luno-mcp", "0.1.0")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// The server URL includes the full URL, but we need to override the base URL
	// Let's access the Luno client and set the base URL directly
	lunoClient := cfg.LunoClient.(*luno.Client)
	lunoClient.SetBaseURL(serverURL)

	// Make a request using the Luno client to see the User-Agent
	fmt.Println("Testing User-Agent header modification...")
	fmt.Printf("Making request to: %s\n", serverURL)
	
	ctx := context.Background()
	_, err = cfg.LunoClient.GetBalances(ctx, nil)
	if err != nil {
		fmt.Printf("Expected error (test server response): %v\n", err)
	}

	fmt.Println("\nTest completed!")
}