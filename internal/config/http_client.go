package config

import (
	"fmt"
	"net/http"
)

// MCPRoundTripper wraps an HTTP RoundTripper to modify User-Agent headers for MCP server identification
type MCPRoundTripper struct {
	transport http.RoundTripper
	mcpServer string
	version   string
}

// NewMCPRoundTripper creates a new RoundTripper wrapper that adds MCP server identification to User-Agent
func NewMCPRoundTripper(transport http.RoundTripper, mcpServer, version string) *MCPRoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &MCPRoundTripper{
		transport: transport,
		mcpServer: mcpServer,
		version:   version,
	}
}

// RoundTrip executes the HTTP request while modifying the User-Agent header to include MCP server identification
func (rt *MCPRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqClone := req.Clone(req.Context())
	
	// Get the current User-Agent header
	currentUA := reqClone.Header.Get("User-Agent")
	
	// Add MCP server identification to the User-Agent
	var newUA string
	if currentUA == "" {
		newUA = fmt.Sprintf("(%s/%s)", rt.mcpServer, rt.version)
	} else {
		newUA = fmt.Sprintf("%s (%s/%s)", currentUA, rt.mcpServer, rt.version)
	}
	reqClone.Header.Set("User-Agent", newUA)
	
	// Execute the request with the modified headers
	return rt.transport.RoundTrip(reqClone)
}