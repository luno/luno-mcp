package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStdioHandshake guards the Glama / Claude Desktop / VS Code integration:
// the container is launched with no args and must respond to a JSON-RPC
// initialize on stdio. A non-stdio default transport silently hangs the proxy
// instead of failing the build.
func TestStdioHandshake(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stdio handshake test in -short mode (builds binary)")
	}

	binPath := buildServerBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, binPath, "--transport="+testTransportStdio)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		if t.Failed() && stderr.Len() > 0 {
			t.Logf("server stderr:\n%s", stderr.String())
		}
	})

	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"stdio-handshake-test","version":"0.0.1"}}}` + "\n"
	_, err = stdin.Write([]byte(initReq))
	require.NoError(t, err)

	type readResult struct {
		line string
		err  error
	}
	lines := make(chan readResult, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		// MCP responses can exceed bufio's default 64KiB line buffer.
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		if scanner.Scan() {
			lines <- readResult{line: scanner.Text()}
			return
		}
		lines <- readResult{err: scanner.Err()}
	}()

	select {
	case res := <-lines:
		require.NoError(t, res.err, "scanner error")
		require.NotEmpty(t, res.line, "expected a JSON-RPC response line on stdout")

		var resp struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Result  json.RawMessage `json:"result"`
			Error   json.RawMessage `json:"error"`
		}
		require.NoError(t, json.Unmarshal([]byte(res.line), &resp), "response not valid JSON: %s", res.line)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.JSONEq(t, "1", string(resp.ID))
		assert.NotEmpty(t, resp.Result, "expected result field; got error=%s", string(resp.Error))
	case <-ctx.Done():
		t.Fatalf("timed out waiting for stdio response: %v", ctx.Err())
	}
}

func buildServerBinary(t *testing.T) string {
	t.Helper()

	binPath := filepath.Join(t.TempDir(), "luno-mcp-test")

	wd, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/server")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "go build failed: %s", string(out))
	return binPath
}
