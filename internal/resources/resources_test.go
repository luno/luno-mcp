package resources

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/luno/luno-go"
	"github.com/luno/luno-go/decimal"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/luno/luno-mcp/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

const (
	expectedMIMEType = "application/json"
	expectedNameFmt  = "Expected name %q, got %q"
)

func TestNewWalletResource(t *testing.T) {
	resource := NewWalletResource()

	assert.Equal(t, WalletResourceURI, resource.URI)
	assert.Equal(t, "Luno Wallets", resource.Name)
	assert.Equal(t, expectedMIMEType, resource.MIMEType)
}

func TestNewTransactionsResource(t *testing.T) {
	resource := NewTransactionsResource()

	assert.Equal(t, TransactionsResourceURI, resource.URI)
	assert.Equal(t, "Luno Transactions", resource.Name)
	assert.Equal(t, expectedMIMEType, resource.MIMEType)
}

func TestNewAccountTemplate(t *testing.T) {
	expectedJSON := `{
		"uriTemplate": "luno://accounts/{id}",
		"name": "Luno Account",
		"description": "Returns details for a specific Luno account"
	}`

	template := NewAccountTemplate()

	actualJSON, err := json.Marshal(template)
	assert.NoError(t, err)

	// Compare JSON structures directly. Can't create an expected object as the fields can only be set internally.
	assert.JSONEq(t, expectedJSON, string(actualJSON))
}

func TestExtractAccountID(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{"valid account URI", "luno://accounts/1234567890", "1234567890"},
		{"empty URI", "", ""},
		{"invalid format", "luno://accounts", ""},
		{"short URI", "luno://", ""},
		{"no account ID", "luno://accounts/", ""},
		{"different resource", "luno://wallets/123", "123"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractAccountID(tc.uri)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestHandleWalletResourceStructure tests that the wallet resource handler can be created and handles nil config
func TestHandleWalletResourceStructure(t *testing.T) {
	handler := HandleWalletResource(nil)
	assert.NotNil(t, handler, "HandleWalletResource should return a non-nil handler")

	// Verify handler returns error with nil config
	req := mcp.ReadResourceRequest{
		Params: struct {
			URI       string         `json:"uri"`
			Arguments map[string]any `json:"arguments,omitempty"`
		}{
			URI: WalletResourceURI,
		},
	}
	result, err := handler(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestHandleTransactionsResourceStructure tests the transactions resource handler structure
func TestHandleTransactionsResourceStructure(t *testing.T) {
	handler := HandleTransactionsResource(nil)
	assert.NotNil(t, handler, "HandleTransactionsResource should return a non-nil handler")
}

// TestHandleAccountTemplateStructure tests the account template handler structure
func TestHandleAccountTemplateStructure(t *testing.T) {
	handler := HandleAccountTemplate(nil)
	assert.NotNil(t, handler, "HandleAccountTemplate should return a non-nil handler")
}

// createTestConfig creates a minimal configuration for testing with a nil Luno client.
// This configuration will cause handlers to return errors when invoked, which is useful
// for testing error handling paths.
func createTestConfig() *config.Config {
	// For testing, we create a config with a nil client
	// In real integration tests, this would be a properly configured client
	return &config.Config{
		LunoClient: nil,
	}
}

// TestHandleWalletResourceIntegration tests the wallet resource handler structure and behavior
func TestHandleWalletResourceIntegration(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name:        "config with nil client",
			config:      createTestConfig(),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := HandleWalletResource(tc.config)
			assert.NotNil(t, handler, "HandleWalletResource should return a non-nil handler")

			req := mcp.ReadResourceRequest{
				Params: struct {
					URI       string         `json:"uri"`
					Arguments map[string]any `json:"arguments,omitempty"`
				}{
					URI: WalletResourceURI,
				},
			}

			result, err := handler(context.Background(), req)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestHandleTransactionsResourceIntegration tests the transactions resource handler structure and behavior
func TestHandleTransactionsResourceIntegration(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
		},
		{
			name:        "config with nil client",
			config:      createTestConfig(),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := HandleTransactionsResource(tc.config)
			assert.NotNil(t, handler, "HandleTransactionsResource should return a non-nil handler")

			req := mcp.ReadResourceRequest{
				Params: struct {
					URI       string         `json:"uri"`
					Arguments map[string]any `json:"arguments,omitempty"`
				}{
					URI: TransactionsResourceURI,
				},
			}

			result, err := handler(context.Background(), req)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestHandleAccountTemplateIntegration tests the account template handler structure and behavior
func TestHandleAccountTemplateIntegration(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		uri         string
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			uri:         "luno://accounts/1234567890",
			expectError: true,
		},
		{
			name:        "config with nil client",
			config:      createTestConfig(),
			uri:         "luno://accounts/1234567890",
			expectError: true,
		},
		{
			name:        "invalid URI format",
			config:      createTestConfig(),
			uri:         "invalid://uri",
			expectError: true,
		},
		{
			name:        "empty account ID",
			config:      createTestConfig(),
			uri:         "luno://accounts/",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := HandleAccountTemplate(tc.config)
			assert.NotNil(t, handler, "HandleAccountTemplate should return a non-nil handler")

			req := mcp.ReadResourceRequest{
				Params: struct {
					URI       string         `json:"uri"`
					Arguments map[string]any `json:"arguments,omitempty"`
				}{
					URI: tc.uri,
				},
			}

			result, err := handler(context.Background(), req)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// newReadResourceRequest is a helper to build mcp.ReadResourceRequest values in tests.
func newReadResourceRequest(uri string) mcp.ReadResourceRequest {
	return mcp.ReadResourceRequest{
		Params: struct {
			URI       string         `json:"uri"`
			Arguments map[string]any `json:"arguments,omitempty"`
		}{
			URI: uri,
		},
	}
}

// newDecimal is a test helper that creates a decimal.Decimal from a string.
func newDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal.NewFromString(%q) failed: %v", s, err)
	}
	return d
}

func TestHandleWalletResourceWithMock(t *testing.T) {
	tests := []struct {
		name          string
		mockSetup     func(*testing.T, *sdk.MockLunoClient)
		expectError   bool
		errorContains string
		checkResult   func(*testing.T, []mcp.ResourceContents)
	}{
		{
			name: "successful wallet retrieval returns JSON with balances",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				resp := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{
							AccountId: "111",
							Asset:     "XBT",
							Balance:   newDecimal(t, "1.5"),
							Reserved:  newDecimal(t, "0.1"),
							Name:      "BTC Account",
						},
					},
				}
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(resp, nil)
			},
			expectError: false,
			checkResult: func(t *testing.T, contents []mcp.ResourceContents) {
				t.Helper()
				assert.Len(t, contents, 1)
				text, ok := contents[0].(mcp.TextResourceContents)
				assert.True(t, ok)
				assert.Equal(t, WalletResourceURI, text.URI)
				assert.Equal(t, expectedMIMEType, text.MIMEType)
				assert.Contains(t, text.Text, "XBT")
			},
		},
		{
			name: "GetBalances API error is propagated",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(nil, errors.New("network error"))
			},
			expectError:   true,
			errorContains: "failed to get balances",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tc.mockSetup(t, mockClient)

			cfg := &config.Config{LunoClient: mockClient}
			handler := HandleWalletResource(cfg)

			result, err := handler(context.Background(), newReadResourceRequest(WalletResourceURI))

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}
		})
	}
}

func TestHandleTransactionsResourceWithMock(t *testing.T) {
	tests := []struct {
		name          string
		mockSetup     func(*testing.T, *sdk.MockLunoClient)
		expectError   bool
		errorContains string
		checkResult   func(*testing.T, []mcp.ResourceContents)
	}{
		{
			name: "empty balances returns empty JSON array",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(&luno.GetBalancesResponse{Balance: []luno.AccountBalance{}}, nil)
			},
			expectError: false,
			checkResult: func(t *testing.T, contents []mcp.ResourceContents) {
				t.Helper()
				assert.Len(t, contents, 1)
				text, ok := contents[0].(mcp.TextResourceContents)
				assert.True(t, ok)
				assert.Equal(t, "[]", text.Text)
			},
		},
		{
			name: "account with non-zero balance fetches transactions",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				balanceResp := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{AccountId: "42", Asset: "XBT", Balance: newDecimal(t, "0.5"), Name: "XBT"},
					},
				}
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(balanceResp, nil)

				txnResp := &luno.ListTransactionsResponse{
					Id:           "42",
					Transactions: []luno.Transaction{},
				}
				m.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id: 42, MinRow: 0, MaxRow: 20,
				}).Return(txnResp, nil)
			},
			expectError: false,
			checkResult: func(t *testing.T, contents []mcp.ResourceContents) {
				t.Helper()
				assert.Len(t, contents, 1)
				text, ok := contents[0].(mcp.TextResourceContents)
				assert.True(t, ok)
				assert.Equal(t, TransactionsResourceURI, text.URI)
				assert.Equal(t, expectedMIMEType, text.MIMEType)
			},
		},
		{
			name: "account with zero balance falls back to first account",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				balanceResp := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{AccountId: "7", Asset: "ZAR", Balance: newDecimal(t, "0"), Name: "ZAR"},
					},
				}
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(balanceResp, nil)

				txnResp := &luno.ListTransactionsResponse{Id: "7", Transactions: []luno.Transaction{}}
				m.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id: 7, MinRow: 0, MaxRow: 20,
				}).Return(txnResp, nil)
			},
			expectError: false,
		},
		{
			name: "GetBalances error is propagated",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(nil, errors.New("API down"))
			},
			expectError:   true,
			errorContains: "failed to get balances",
		},
		{
			name: "ListTransactions error is propagated",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				balanceResp := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{AccountId: "99", Asset: "ETH", Balance: newDecimal(t, "2.0"), Name: "ETH"},
					},
				}
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(balanceResp, nil)
				m.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id: 99, MinRow: 0, MaxRow: 20,
				}).Return(nil, errors.New("txn error"))
			},
			expectError:   true,
			errorContains: "failed to get transactions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tc.mockSetup(t, mockClient)

			cfg := &config.Config{LunoClient: mockClient}
			handler := HandleTransactionsResource(cfg)

			result, err := handler(context.Background(), newReadResourceRequest(TransactionsResourceURI))

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}
		})
	}
}

func TestHandleAccountTemplateWithMock(t *testing.T) {
	tests := []struct {
		name          string
		uri           string
		mockSetup     func(*testing.T, *sdk.MockLunoClient)
		expectError   bool
		errorContains string
		checkResult   func(*testing.T, []mcp.ResourceContents)
	}{
		{
			name: "successful account retrieval with matching balance",
			uri:  "luno://accounts/123",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				balanceResp := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{AccountId: "123", Asset: "XBT", Balance: newDecimal(t, "0.5"), Name: "XBT"},
					},
				}
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(balanceResp, nil)

				txnResp := &luno.ListTransactionsResponse{Id: "123", Transactions: []luno.Transaction{}}
				m.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id: 123, MinRow: 0, MaxRow: 10,
				}).Return(txnResp, nil)
			},
			expectError: false,
			checkResult: func(t *testing.T, contents []mcp.ResourceContents) {
				t.Helper()
				assert.Len(t, contents, 1)
				text, ok := contents[0].(mcp.TextResourceContents)
				assert.True(t, ok)
				assert.Equal(t, "luno://accounts/123", text.URI)
				assert.Equal(t, expectedMIMEType, text.MIMEType)

				// Verify the combined result contains both account and transactions keys.
				// This exercises the map[string]any introduced in the PR.
				var result map[string]any
				assert.NoError(t, json.Unmarshal([]byte(text.Text), &result))
				assert.Contains(t, result, "account")
				assert.Contains(t, result, "transactions")
			},
		},
		{
			name: "account not found in balances still fetches transactions",
			uri:  "luno://accounts/456",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				balanceResp := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{AccountId: "999", Asset: "ZAR", Balance: newDecimal(t, "0"), Name: "ZAR"},
					},
				}
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(balanceResp, nil)

				txnResp := &luno.ListTransactionsResponse{Id: "456", Transactions: []luno.Transaction{}}
				m.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id: 456, MinRow: 0, MaxRow: 10,
				}).Return(txnResp, nil)
			},
			expectError: false,
		},
		{
			name: "GetBalances error is propagated",
			uri:  "luno://accounts/123",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(nil, errors.New("balances API down"))
			},
			expectError:   true,
			errorContains: "failed to get account details",
		},
		{
			name: "ListTransactions error is propagated",
			uri:  "luno://accounts/789",
			mockSetup: func(t *testing.T, m *sdk.MockLunoClient) {
				balanceResp := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{AccountId: "789", Asset: "ETH", Balance: newDecimal(t, "1.0"), Name: "ETH"},
					},
				}
				m.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(balanceResp, nil)
				m.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id: 789, MinRow: 0, MaxRow: 10,
				}).Return(nil, errors.New("txn fetch failed"))
			},
			expectError:   true,
			errorContains: "failed to get transactions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tc.mockSetup(t, mockClient)

			cfg := &config.Config{LunoClient: mockClient}
			handler := HandleAccountTemplate(cfg)

			result, err := handler(context.Background(), newReadResourceRequest(tc.uri))

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tc.checkResult != nil {
					tc.checkResult(t, result)
				}
			}
		})
	}
}
