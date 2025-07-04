package config

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPRoundTripper(t *testing.T) {
	tests := []struct {
		name            string
		mcpServer       string
		version         string
		originalUA      string
		expectedUA      string
		transport       http.RoundTripper
	}{
		{
			name:       "adds MCP identification to existing User-Agent",
			mcpServer:  "luno-mcp",
			version:    "1.0.0",
			originalUA: "LunoGoSDK/0.0.34 go1.24 linux amd64",
			expectedUA: "LunoGoSDK/0.0.34 go1.24 linux amd64 (luno-mcp/1.0.0)",
			transport:  http.DefaultTransport,
		},
		{
			name:       "adds MCP identification to empty User-Agent",
			mcpServer:  "test-app",
			version:    "2.0.0",
			originalUA: "",
			expectedUA: "(test-app/2.0.0)",
			transport:  http.DefaultTransport,
		},
		{
			name:       "works with nil transport",
			mcpServer:  "luno-mcp",
			version:    "1.0.0",
			originalUA: "TestClient/1.0",
			expectedUA: "TestClient/1.0 (luno-mcp/1.0.0)",
			transport:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server that captures the User-Agent header
			var capturedUA string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedUA = r.Header.Get("User-Agent")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create MCP round tripper
			mcpRT := NewMCPRoundTripper(tc.transport, tc.mcpServer, tc.version)

			// Create HTTP client with our round tripper
			client := &http.Client{Transport: mcpRT}

			// Create request with original User-Agent
			req, err := http.NewRequest("GET", server.URL, nil)
			require.NoError(t, err)
			
			if tc.originalUA != "" {
				req.Header.Set("User-Agent", tc.originalUA)
			}

			// Execute request
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify the User-Agent header was modified correctly
			assert.Equal(t, tc.expectedUA, capturedUA)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})
	}
}

func TestNewMCPRoundTripper(t *testing.T) {
	t.Run("uses default transport when nil is provided", func(t *testing.T) {
		rt := NewMCPRoundTripper(nil, "test-app", "1.0.0")
		assert.NotNil(t, rt)
		assert.Equal(t, http.DefaultTransport, rt.transport)
		assert.Equal(t, "test-app", rt.mcpServer)
		assert.Equal(t, "1.0.0", rt.version)
	})

	t.Run("uses provided transport", func(t *testing.T) {
		customTransport := &http.Transport{}
		rt := NewMCPRoundTripper(customTransport, "luno-mcp", "2.0.0")
		assert.NotNil(t, rt)
		assert.Equal(t, customTransport, rt.transport)
		assert.Equal(t, "luno-mcp", rt.mcpServer)
		assert.Equal(t, "2.0.0", rt.version)
	})
}

func TestMCPRoundTripperRequestCloning(t *testing.T) {
	t.Run("does not modify original request", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create MCP round tripper
		mcpRT := NewMCPRoundTripper(nil, "luno-mcp", "1.0.0")

		// Create original request
		originalUA := "OriginalClient/1.0"
		req, err := http.NewRequest("GET", server.URL, nil)
		require.NoError(t, err)
		req.Header.Set("User-Agent", originalUA)

		// Execute request
		_, err = mcpRT.RoundTrip(req)
		require.NoError(t, err)

		// Verify original request was not modified
		assert.Equal(t, originalUA, req.Header.Get("User-Agent"))
	})
}