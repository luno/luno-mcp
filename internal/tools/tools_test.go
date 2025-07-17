package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/luno/luno-go"
	"github.com/luno/luno-go/decimal"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/luno/luno-mcp/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// NewFromString is a test helper that creates a decimal from a string, failing the test on error.
func NewFromString(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("NewFromString(%q) failed: %v", s, err)
	}
	return d
}

// NewFromFloat64 is a test helper that creates a decimal from a float64, failing the test on error.
func NewFromFloat64(t *testing.T, f float64) decimal.Decimal {
	t.Helper()
	d := decimal.NewFromFloat64(f, 8)
	return d
}

const (
	apiErrorStr               = "API error"
	missingPairParameterStr   = "missing pair parameter"
	gettingPairFromRequestStr = "getting pair from request"
	invalidPairStr            = "invalid pair"
	testTimestamp             = 1640995200000 // January 1, 2022 00:00:00 UTC
)

func TestNormalizeCurrencyPair(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple BTC to XBT", "BTC", "XBT"},
		{"BTC in pair", "BTCGBP", "XBTGBP"},
		{"BTC with hyphen separator", "BTC-GBP", "XBTGBP"},
		{"BTC with slash separator", "BTC/GBP", "XBTGBP"},
		{"BTC with underscore separator", "BTC_GBP", "XBTGBP"},
		{"Lowercase input", "btcgbp", "XBTGBP"},
		{"Mixed case input", "xbTGbP", "XBTGBP"},
		{"Non-BTC pair", "ETHZAR", "ETHZAR"},
		{"Non-BTC pair with separator", "ETH-ZAR", "ETHZAR"},
		{"BITCOIN text conversion", "BITCOIN", "XBT"},
		{"BITCOIN in pair", "BITCOINUSD", "XBTUSD"},
		{"Multiple separators", "BTC-_/GBP", "XBTGBP"},
		{"Combo of mappings", "BITCOIN/GBP", "XBTGBP"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeCurrencyPair(tc.input)
			if result != tc.expected {
				t.Errorf("normalizeCurrencyPair(%q) = %q, want %q",
					tc.input, result, tc.expected)
			}
		})
	}
}

func TestToolCreation(t *testing.T) {
	tests := []struct {
		name     string
		toolFunc func() mcp.Tool
		toolName string
		params   []string
	}{
		{
			name:     "GetBalances tool",
			toolFunc: NewGetBalancesTool,
			toolName: GetBalancesToolID,
			params:   []string{},
		},
		{
			name:     "GetTicker tool",
			toolFunc: NewGetTickerTool,
			toolName: GetTickerToolID,
			params:   []string{"pair"},
		},
		{
			name:     "GetOrderBook tool",
			toolFunc: NewGetOrderBookTool,
			toolName: GetOrderBookToolID,
			params:   []string{"pair"},
		},
		{
			name:     "CreateOrder tool",
			toolFunc: NewCreateOrderTool,
			toolName: CreateOrderToolID,
			params:   []string{"pair", "type", "volume", "price"},
		},
		{
			name:     "CancelOrder tool",
			toolFunc: NewCancelOrderTool,
			toolName: CancelOrderToolID,
			params:   []string{"order_id"},
		},
		{
			name:     "ListOrders tool",
			toolFunc: NewListOrdersTool,
			toolName: ListOrdersToolID,
			params:   []string{"pair", "limit"},
		},
		{
			name:     "ListTransactions tool",
			toolFunc: NewListTransactionsTool,
			toolName: ListTransactionsToolID,
			params:   []string{"account_id", "min_row", "max_row"},
		},
		{
			name:     "GetTransaction tool",
			toolFunc: NewGetTransactionTool,
			toolName: GetTransactionToolID,
			params:   []string{"account_id", "transaction_id"},
		},
		{
			name:     "ListTrades tool",
			toolFunc: NewListTradesTool,
			toolName: ListTradesToolID,
			params:   []string{"pair", "since"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tool := tt.toolFunc()

			if tool.Name != tt.toolName {
				t.Errorf("Expected tool name %q, got %q", tt.toolName, tool.Name)
			}

			if tool.Description == "" {
				t.Error("Tool description should not be empty")
			}

			// Verify tool has proper schema structure
			if tool.InputSchema.Type == "" {
				t.Error("Tool should have an input schema type")
				return
			}

			// Verify expected parameters exist
			if tool.InputSchema.Properties == nil {
				if len(tt.params) > 0 {
					t.Error("Tool should have properties for parameters")
				}
				return
			}

			for _, param := range tt.params {
				if _, exists := tool.InputSchema.Properties[param]; !exists {
					t.Errorf("Expected parameter %q to exist", param)
				}
			}
		})
	}
}

// Helper function to extract text content from mcp.CallToolResult
func getTextContentFromResult(t *testing.T, result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		t.Fatal("result or content is nil or empty")
	}
	// Extract text content from the first content item.
	// This assumes single text content, which is the current pattern.
	if len(result.Content) > 1 {
		t.Fatalf("expected only one content item, got multiple")
	}
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		return textContent.Text
	}
	t.Fatalf("expected mcp.TextContent but got %T", result.Content[0])
	return "" // Should not be reached
}

func TestHandleGetBalances(t *testing.T) {
	tests := []struct {
		name            string
		mockSetup       func(*testing.T, *sdk.MockLunoClient)
		isAuthenticated bool
		expectedError   bool
		errorContains   string
	}{
		{
			name: "successful get balances",
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				balance1 := NewFromString(t, "1.5")
				reserved1 := NewFromString(t, "0.1")
				unconfirmed1 := NewFromString(t, "0.0")
				balance2 := NewFromString(t, "10000.0")
				reserved2 := NewFromString(t, "0.0")
				unconfirmed2 := NewFromString(t, "0.0")

				mockResponse := &luno.GetBalancesResponse{
					Balance: []luno.AccountBalance{
						{
							AccountId:   "123456",
							Asset:       "XBT",
							Balance:     balance1,
							Reserved:    reserved1,
							Unconfirmed: unconfirmed1,
							Name:        "XBT Account",
						},
						{
							AccountId:   "789012",
							Asset:       "ZAR",
							Balance:     balance2,
							Reserved:    reserved2,
							Unconfirmed: unconfirmed2,
							Name:        "ZAR Account",
						},
					},
				}
				mockClient.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   false,
		},
		{
			name: "GetBalances API error",
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().GetBalances(context.Background(), &luno.GetBalancesRequest{}).
					Return(nil, errors.New(apiErrorStr))
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Failed to get balances",
		},
		{
			name:            "unauthenticated get balances",
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) {},
			isAuthenticated: false,
			expectedError:   true,
			errorContains:   ErrAPICredentialsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: tt.isAuthenticated,
			}

			handler := HandleGetBalances(cfg)
			request := createMockRequest(nil)

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorText := getTextContentFromResult(t, result)
					assert.Contains(t, errorText, tt.errorContains)
				}
			} else {
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)

				// Verify JSON structure
				var balances []map[string]any
				err := json.Unmarshal([]byte(textContent), &balances)
				assert.NoError(t, err)
				assert.Len(t, balances, 2, "Should have 2 balances")
			}
		})
	}
}

func TestHandleGetTicker(t *testing.T) {
	tests := []struct {
		name          string
		requestParams map[string]any
		mockSetup     func(*testing.T, *sdk.MockLunoClient)
		expectedError bool
		errorContains string
	}{
		{
			name: "successful get ticker",
			requestParams: map[string]any{
				"pair": "XBTZAR",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.GetTickerResponse{
					Pair:                "XBTZAR",
					Timestamp:           luno.Time(time.UnixMilli(testTimestamp)),
					Bid:                 decimal.NewFromInt64(800000),
					Ask:                 decimal.NewFromInt64(800100),
					LastTrade:           decimal.NewFromInt64(800050),
					Rolling24HourVolume: decimal.NewFromFloat64(100.5, -1),
					Status:              "ACTIVE",
				}
				mockClient.EXPECT().GetTicker(context.Background(), &luno.GetTickerRequest{Pair: "XBTZAR"}).
					Return(mockResponse, nil)
			},
			expectedError: false,
		},
		{
			name: "BTC to XBT normalization",
			requestParams: map[string]any{
				"pair": "BTCZAR",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.GetTickerResponse{
					Pair:                "XBTZAR",
					Timestamp:           luno.Time(time.UnixMilli(testTimestamp)),
					Bid:                 decimal.NewFromFloat64(800000, -1),
					Ask:                 decimal.NewFromFloat64(800100, -1),
					LastTrade:           decimal.NewFromFloat64(800050, -1),
					Rolling24HourVolume: decimal.NewFromFloat64(100.5, -1),
					Status:              "ACTIVE",
				}
				mockClient.EXPECT().GetTicker(context.Background(), &luno.GetTickerRequest{Pair: "XBTZAR"}).
					Return(mockResponse, nil)
			},
			expectedError: false,
		},
		{
			name:          "missing pair for getTicker",
			requestParams: map[string]any{},
			mockSetup:     func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed for this case */ },
			expectedError: true,
			errorContains: gettingPairFromRequestStr,
		},
		{
			name: "GetTicker API error",
			requestParams: map[string]any{
				"pair": "INVALID",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().GetTicker(context.Background(), &luno.GetTickerRequest{Pair: "INVALID"}).
					Return(nil, errors.New(invalidPairStr))
			},
			expectedError: true,
			errorContains: "getting ticker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: true, // Public endpoint, but setting to true for consistency
			}

			handler := HandleGetTicker(cfg)
			request := createMockRequest(tt.requestParams)

			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorText := getTextContentFromResult(t, result)
					assert.Contains(t, errorText, tt.errorContains)
				}
			} else {
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)

				// Verify JSON structure
				var ticker map[string]any
				err := json.Unmarshal([]byte(textContent), &ticker)
				assert.NoError(t, err)
				assert.Equal(t, "XBTZAR", ticker["pair"])
			}
		})
	}
}

func TestHandleGetOrderBook(t *testing.T) {
	tests := []struct {
		name          string
		requestParams map[string]any
		mockSetup     func(*testing.T, *sdk.MockLunoClient)
		expectedError bool
		errorContains string
	}{
		{
			name: "successful get order book",
			requestParams: map[string]any{
				"pair": "XBTZAR",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.GetOrderBookResponse{
					Timestamp: testTimestamp,
					Bids: []luno.OrderBookEntry{
						{Price: decimal.NewFromInt64(800000), Volume: decimal.NewFromFloat64(0.5, -1)},
						{Price: decimal.NewFromInt64(799900), Volume: decimal.NewFromFloat64(1.0, -1)},
					},
					Asks: []luno.OrderBookEntry{
						{Price: decimal.NewFromInt64(800100), Volume: decimal.NewFromFloat64(0.8, -1)},
						{Price: decimal.NewFromInt64(800200), Volume: decimal.NewFromFloat64(1.2, -1)},
					},
				}
				mockClient.EXPECT().GetOrderBook(context.Background(), &luno.GetOrderBookRequest{Pair: "XBTZAR"}).
					Return(mockResponse, nil)
			},
			expectedError: false,
		},
		{
			name:          "missing pair for GetOrderBook",
			requestParams: map[string]any{},
			mockSetup:     func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed for this case */ },
			expectedError: true,
			errorContains: gettingPairFromRequestStr,
		},
		{
			name: "GetOrderBook API error",
			requestParams: map[string]any{
				"pair": "INVALID",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().GetOrderBook(context.Background(), &luno.GetOrderBookRequest{Pair: "INVALID"}).
					Return(nil, errors.New(invalidPairStr))
			},
			expectedError: true,
			errorContains: "getting order book",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: true, // Public endpoint
			}

			handler := HandleGetOrderBook(cfg)
			request := createMockRequest(tt.requestParams)

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorMsg := getTextContentFromResult(t, result)
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)

				// Verify JSON structure
				var orderBook map[string]any
				err := json.Unmarshal([]byte(textContent), &orderBook)
				assert.NoError(t, err)
				assert.Contains(t, orderBook, "bids")
				assert.Contains(t, orderBook, "asks")
			}
		})
	}
}

func TestHandleCancelOrder(t *testing.T) {
	tests := []struct {
		name            string
		requestParams   map[string]any
		mockSetup       func(*testing.T, *sdk.MockLunoClient)
		isAuthenticated bool
		expectedError   bool
		errorContains   string
	}{
		{
			name: "successful cancel order",
			requestParams: map[string]any{
				"order_id": "12345",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.StopOrderResponse{
					Success: true,
				}
				mockClient.EXPECT().StopOrder(context.Background(), &luno.StopOrderRequest{OrderId: "12345"}).
					Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   false,
		},
		{
			name:            "missing order_id parameter",
			requestParams:   map[string]any{},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed for this case */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "getting order_id from request",
		},
		{
			name: "CancelOrder API error",
			requestParams: map[string]any{
				"order_id": "invalid_id",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().StopOrder(context.Background(), &luno.StopOrderRequest{OrderId: "invalid_id"}).
					Return(nil, errors.New("Order not found"))
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Failed to cancel order",
		},
		{
			name:            "unauthenticated cancel order",
			requestParams:   map[string]any{"order_id": "12345"},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) {},
			isAuthenticated: false,
			expectedError:   true,
			errorContains:   ErrAPICredentialsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: tt.isAuthenticated,
			}

			handler := HandleCancelOrder(cfg)
			request := createMockRequest(tt.requestParams)

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorMsg := getTextContentFromResult(t, result)
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)
			}
		})
	}
}

func TestHandleListOrders(t *testing.T) {
	tests := []struct {
		name            string
		requestParams   map[string]any
		mockSetup       func(*testing.T, *sdk.MockLunoClient)
		isAuthenticated bool
		expectedError   bool
		errorContains   string
	}{
		{
			name: "successful list orders with pair",
			requestParams: map[string]any{
				"pair":  "XBTZAR",
				"limit": float64(50),
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.ListOrdersResponse{
					Orders: []luno.Order{
						{
							OrderId:             "12345",
							CreationTimestamp:   luno.Time(time.UnixMilli(testTimestamp)),
							ExpirationTimestamp: luno.Time(time.UnixMilli(testTimestamp + 86400000)),
							CompletedTimestamp:  luno.Time(time.UnixMilli(0)),
							Type:                luno.OrderTypeBid,
							State:               luno.OrderStatePending,
							LimitPrice:          decimal.NewFromInt64(800000),
							LimitVolume:         decimal.NewFromFloat64(0.001, -1),
							Base:                decimal.NewFromFloat64(0.0, -1),
							Counter:             decimal.NewFromFloat64(0.0, -1),
							FeeBase:             decimal.NewFromFloat64(0.0, -1),
							FeeCounter:          decimal.NewFromFloat64(0.0, -1),
							Pair:                "XBTZAR",
						},
					},
				}
				mockClient.EXPECT().ListOrders(context.Background(), &luno.ListOrdersRequest{
					Pair:  "XBTZAR",
					Limit: 50,
				}).Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   false,
		},
		{
			name:          "successful list orders without pair",
			requestParams: map[string]any{},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.ListOrdersResponse{
					Orders: []luno.Order{},
				}
				mockClient.EXPECT().ListOrders(context.Background(), &luno.ListOrdersRequest{
					Pair:  "",
					Limit: 100, // Default limit
				}).Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   false,
		},
		{
			name: "ListOrders API error",
			requestParams: map[string]any{
				"pair": "INVALID",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().ListOrders(context.Background(), &luno.ListOrdersRequest{
					Pair:  "INVALID",
					Limit: 100, // Default limit
				}).Return(nil, errors.New(invalidPairStr))
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Failed to list orders",
		},
		{
			name:            "unauthenticated list orders",
			requestParams:   map[string]any{},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) {},
			isAuthenticated: false,
			expectedError:   true,
			errorContains:   ErrAPICredentialsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: tt.isAuthenticated,
			}

			handler := HandleListOrders(cfg)
			request := createMockRequest(tt.requestParams)

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorMsg := getTextContentFromResult(t, result)
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				assert.False(t, result.IsError)
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)

				// Verify JSON structure
				var ordersResponse map[string]any
				err := json.Unmarshal([]byte(textContent), &ordersResponse)
				assert.NoError(t, err)
				assert.Contains(t, ordersResponse, "orders")
			}
		})
	}
}

func TestHandleListTransactions(t *testing.T) {
	tests := []struct {
		name            string
		requestParams   map[string]any
		mockSetup       func(*testing.T, *sdk.MockLunoClient)
		isAuthenticated bool
		expectedError   bool
		errorContains   string
	}{
		{
			name: "successful list transactions",
			requestParams: map[string]any{
				"account_id": "123456",
				"min_row":    float64(1),
				"max_row":    float64(10),
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.ListTransactionsResponse{
					Id: "123456",
					Transactions: []luno.Transaction{
						{
							RowIndex:       1,
							Timestamp:      luno.Time(time.UnixMilli(testTimestamp)),
							Balance:        decimal.NewFromFloat64(1.5, -1),
							Available:      decimal.NewFromFloat64(1.4, -1),
							AvailableDelta: decimal.NewFromFloat64(0.1, -1),
							BalanceDelta:   decimal.NewFromFloat64(0.1, -1),
							Currency:       "XBT",
							Description:    "Test transaction",
						},
					},
				}
				// Convert account_id from string to int64 for the request
				accountIdInt, _ := strconv.ParseInt("123456", 10, 64)
				mockClient.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id:     accountIdInt,
					MinRow: 1,
					MaxRow: 10,
				}).Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   false,
		},
		{
			name:            "missing account_id parameter",
			requestParams:   map[string]any{},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed for this case */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "getting account_id from request",
		},
		{
			name: "invalid account_id format",
			requestParams: map[string]any{
				"account_id": "not_a_number",
			},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed for this case */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Invalid account ID format",
		},
		{
			name: "ListTransactions API error",
			requestParams: map[string]any{
				"account_id": "999999",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				accountIdInt, _ := strconv.ParseInt("999999", 10, 64)
				mockClient.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id:     accountIdInt,
					MinRow: 1,   // Default min_row
					MaxRow: 100, // Default max_row
				}).Return(nil, errors.New("Account not found"))
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Failed to list transactions",
		},
		{
			name:            "unauthenticated list transactions",
			requestParams:   map[string]any{"account_id": "123456"},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) {},
			isAuthenticated: false,
			expectedError:   true,
			errorContains:   ErrAPICredentialsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: tt.isAuthenticated,
			}

			handler := HandleListTransactions(cfg)
			request := createMockRequest(tt.requestParams)

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorMsg := getTextContentFromResult(t, result)
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				assert.False(t, result.IsError)
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)

				// Verify JSON structure
				var transactionsResponse map[string]any
				err := json.Unmarshal([]byte(textContent), &transactionsResponse)
				assert.NoError(t, err)
				assert.Contains(t, transactionsResponse, "transactions")
			}
		})
	}
}

func TestHandleGetTransaction(t *testing.T) {
	tests := []struct {
		name            string
		requestParams   map[string]any
		mockSetup       func(*testing.T, *sdk.MockLunoClient)
		isAuthenticated bool
		expectedError   bool
		errorContains   string
	}{
		{
			name: "successful get transaction",
			requestParams: map[string]any{
				"account_id":     "123456",
				"transaction_id": "5",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.ListTransactionsResponse{
					Id: "123456",
					Transactions: []luno.Transaction{
						{
							RowIndex:       5,
							Timestamp:      luno.Time(time.UnixMilli(testTimestamp)),
							Balance:        decimal.NewFromFloat64(1.5, -1),
							Available:      decimal.NewFromFloat64(1.4, -1),
							AvailableDelta: decimal.NewFromFloat64(0.1, -1),
							BalanceDelta:   decimal.NewFromFloat64(0.1, -1),
							Currency:       "XBT",
							Description:    "Target transaction",
						},
						{
							RowIndex:       6,
							Timestamp:      luno.Time(time.UnixMilli(testTimestamp + 100000)),
							Balance:        decimal.NewFromFloat64(1.6, -1),
							Available:      decimal.NewFromFloat64(1.5, -1),
							AvailableDelta: decimal.NewFromFloat64(0.1, -1),
							BalanceDelta:   decimal.NewFromFloat64(0.1, -1),
							Currency:       "XBT",
							Description:    "Another transaction",
						},
					},
				}
				accountIdInt, _ := strconv.ParseInt("123456", 10, 64)
				mockClient.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id:     accountIdInt,
					MinRow: 0,    // Default min_row for GetTransaction
					MaxRow: 1000, // Default max_row for GetTransaction
				}).Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   false,
		},
		{
			name: "transaction not found",
			requestParams: map[string]any{
				"account_id":     "123456",
				"transaction_id": "999",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.ListTransactionsResponse{
					Id:           "123456",
					Transactions: []luno.Transaction{},
				}
				accountIdInt, _ := strconv.ParseInt("123456", 10, 64)
				mockClient.EXPECT().ListTransactions(context.Background(), &luno.ListTransactionsRequest{
					Id:     accountIdInt,
					MinRow: 0,
					MaxRow: 1000,
				}).Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Transaction not found",
		},
		{
			name: "missing account_id parameter",
			requestParams: map[string]any{
				"transaction_id": "5",
			},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "getting account_id from request",
		},
		{
			name: "missing transaction_id parameter",
			requestParams: map[string]any{
				"account_id": "123456",
			},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "getting transaction_id from request",
		},
		{
			name: "invalid account_id format",
			requestParams: map[string]any{
				"account_id":     "not_a_number",
				"transaction_id": "5",
			},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Invalid account ID format",
		},
		{
			name: "invalid transaction_id format",
			requestParams: map[string]any{
				"account_id":     "123456",
				"transaction_id": "not_a_number",
			},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Invalid transaction ID format",
		},
		{
			name:            "unauthenticated get transaction",
			requestParams:   map[string]any{"account_id": "123456", "transaction_id": "5"},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) {},
			isAuthenticated: false,
			expectedError:   true,
			errorContains:   ErrAPICredentialsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: tt.isAuthenticated,
			}

			handler := HandleGetTransaction(cfg)
			request := createMockRequest(tt.requestParams)

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorMsg := getTextContentFromResult(t, result)
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				assert.False(t, result.IsError)
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)

				// Verify JSON structure
				var transaction map[string]any
				err := json.Unmarshal([]byte(textContent), &transaction)
				assert.NoError(t, err)
				assert.Equal(t, float64(5), transaction["row_index"]) // Ensure correct transaction is returned
			}
		})
	}
}

func TestHandleListTrades(t *testing.T) {
	tests := []struct {
		name          string
		requestParams map[string]any
		mockSetup     func(*testing.T, *sdk.MockLunoClient)
		expectedError bool
		errorContains string
	}{
		{
			name: "successful list trades without since",
			requestParams: map[string]any{
				"pair": "XBTZAR",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockResponse := &luno.ListTradesResponse{
					Trades: []luno.PublicTrade{
						{
							Sequence:  123456,
							Timestamp: luno.Time(time.UnixMilli(testTimestamp)),
							Price:     decimal.NewFromInt64(800000),
							Volume:    decimal.NewFromFloat64(0.001, -1),
							IsBuy:     true,
						},
					},
				}
				mockClient.EXPECT().ListTrades(context.Background(), &luno.ListTradesRequest{
					Pair: "XBTZAR",
				}).Return(mockResponse, nil)
			},
			expectedError: false,
		},
		{
			name: "successful list trades with since",
			requestParams: map[string]any{
				"pair":  "XBTZAR",
				"since": strconv.FormatInt(testTimestamp, 10),
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				sinceTime := luno.Time(time.UnixMilli(testTimestamp))
				mockResponse := &luno.ListTradesResponse{
					Trades: []luno.PublicTrade{
						{
							Sequence:  123457,
							Timestamp: luno.Time(time.UnixMilli(testTimestamp + 60000)),
							Price:     decimal.NewFromFloat64(800100, -1),
							Volume:    decimal.NewFromFloat64(0.002, -1),
							IsBuy:     false,
						},
					},
				}
				mockClient.EXPECT().ListTrades(context.Background(), &luno.ListTradesRequest{
					Pair:  "XBTZAR",
					Since: sinceTime,
				}).Return(mockResponse, nil)
			},
			expectedError: false,
		},
		{
			name:          missingPairParameterStr,
			requestParams: map[string]any{},
			mockSetup:     func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			expectedError: true,
			errorContains: gettingPairFromRequestStr,
		},
		{
			name: "invalid since format",
			requestParams: map[string]any{
				"pair":  "XBTZAR",
				"since": "not_a_number",
			},
			mockSetup:     func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			expectedError: true,
			errorContains: "Invalid 'since' timestamp format",
		},
		{
			name: "ListTrades API error",
			requestParams: map[string]any{
				"pair": "INVALID",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().ListTrades(context.Background(), &luno.ListTradesRequest{
					Pair: "INVALID",
				}).Return(nil, errors.New(invalidPairStr))
			},
			expectedError: true,
			errorContains: "listing trades",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient: mockClient,
			}

			handler := HandleListTrades(cfg)
			request := createMockRequest(tt.requestParams)

			result, err := handler(context.Background(), request)
			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorMsg := getTextContentFromResult(t, result)
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				assert.False(t, result.IsError)
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)

				// Verify JSON structure
				var tradesResponse map[string]any
				err := json.Unmarshal([]byte(textContent), &tradesResponse)
				assert.NoError(t, err)
				assert.Contains(t, tradesResponse, "trades")
			}
		})
	}
}

// Helper function to create mock MCP requests
func createMockRequest(params map[string]any) mcp.CallToolRequest {
	arguments := make(map[string]any)
	for k, v := range params {
		arguments[k] = v
	}

	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "test_tool",
			Arguments: arguments,
		},
	}
}

func TestHandleCreateOrder(t *testing.T) {
	tests := []struct {
		name            string
		requestParams   map[string]any
		mockSetup       func(*testing.T, *sdk.MockLunoClient)
		isAuthenticated bool
		expectedError   bool
		errorContains   string
	}{
		{
			name: "successful create order",
			requestParams: map[string]any{
				"pair":   "XBTZAR",
				"type":   "BUY",
				"volume": "0.01",
				"price":  "1000000",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				vol := NewFromString(t, "0.01")
				price := NewFromString(t, "1000000")

				// Mock GetTicker call from GetMarketInfo
				mockTickerResponse := &luno.GetTickerResponse{
					Pair:                "XBTZAR",
					Timestamp:           luno.Time(time.UnixMilli(testTimestamp)),
					Bid:                 decimal.NewFromInt64(800000),
					Ask:                 decimal.NewFromInt64(800100),
					LastTrade:           decimal.NewFromInt64(800050),
					Rolling24HourVolume: decimal.NewFromFloat64(100.5, -1),
					Status:              "ACTIVE",
				}
				mockClient.EXPECT().GetTicker(context.Background(), &luno.GetTickerRequest{Pair: "XBTZAR"}).
					Return(mockTickerResponse, nil)

				// Mock GetOrderBook call from GetMarketInfo
				mockOrderBookResponse := &luno.GetOrderBookResponse{
					Timestamp: testTimestamp,
					Bids: []luno.OrderBookEntry{
						{Price: decimal.NewFromInt64(800000), Volume: decimal.NewFromFloat64(0.5, -1)},
					},
					Asks: []luno.OrderBookEntry{
						{Price: decimal.NewFromInt64(800100), Volume: decimal.NewFromFloat64(0.8, -1)},
					},
				}
				mockClient.EXPECT().GetOrderBook(context.Background(), &luno.GetOrderBookRequest{Pair: "XBTZAR"}).
					Return(mockOrderBookResponse, nil)

				// Mock PostLimitOrder call
				mockResponse := &luno.PostLimitOrderResponse{
					OrderId: "BXMC2SEAS4KF5S2",
				}
				mockClient.EXPECT().PostLimitOrder(context.Background(), &luno.PostLimitOrderRequest{
					Pair:   "XBTZAR",
					Type:   luno.OrderTypeBid,
					Volume: vol,
					Price:  price,
				}).Return(mockResponse, nil)
			},
			isAuthenticated: true,
			expectedError:   false,
		},
		{
			name: "CreateOrder PostLimitOrder API error",
			requestParams: map[string]any{
				"pair":   "XBTZAR",
				"type":   "BUY",
				"volume": "0.01",
				"price":  "1000000",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				vol := NewFromString(t, "0.01")
				price := NewFromString(t, "1000000")

				// Mock GetTicker call from GetMarketInfo
				mockTickerResponse := &luno.GetTickerResponse{
					Pair:                "XBTZAR",
					Timestamp:           luno.Time(time.UnixMilli(testTimestamp)),
					Bid:                 decimal.NewFromInt64(800000),
					Ask:                 decimal.NewFromInt64(800100),
					LastTrade:           decimal.NewFromInt64(800050),
					Rolling24HourVolume: decimal.NewFromFloat64(100.5, -1),
					Status:              "ACTIVE",
				}
				mockClient.EXPECT().GetTicker(context.Background(), &luno.GetTickerRequest{Pair: "XBTZAR"}).
					Return(mockTickerResponse, nil)

				// Mock GetOrderBook call from GetMarketInfo
				mockOrderBookResponse := &luno.GetOrderBookResponse{
					Timestamp: testTimestamp,
					Bids: []luno.OrderBookEntry{
						{Price: decimal.NewFromInt64(800000), Volume: decimal.NewFromFloat64(0.5, -1)},
					},
					Asks: []luno.OrderBookEntry{
						{Price: decimal.NewFromInt64(800100), Volume: decimal.NewFromFloat64(0.8, -1)},
					},
				}
				mockClient.EXPECT().GetOrderBook(context.Background(), &luno.GetOrderBookRequest{Pair: "XBTZAR"}).
					Return(mockOrderBookResponse, nil)

				// Mock PostLimitOrder call that returns error
				mockClient.EXPECT().PostLimitOrder(context.Background(), &luno.PostLimitOrderRequest{
					Pair:   "XBTZAR",
					Type:   luno.OrderTypeBid,
					Volume: vol,
					Price:  price,
				}).Return(nil, errors.New(apiErrorStr))
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Failed to create limit order",
		},
		{
			name: "CreateOrder GetTicker API error",
			requestParams: map[string]any{
				"pair":   "XBTZAR",
				"type":   "BUY",
				"volume": "0.01",
				"price":  "1000000",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().GetTicker(mock.Anything, mock.Anything).Return(nil, errors.New("API error"))
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Unable to create order: Failed to retrieve market information for pair XBTZAR",
		},
		{
			name: "CreateOrder GetOrderBook API error",
			requestParams: map[string]any{
				"pair":   "XBTZAR",
				"type":   "BUY",
				"volume": "0.01",
				"price":  "1000000",
			},
			mockSetup: func(t *testing.T, mockClient *sdk.MockLunoClient) {
				mockClient.EXPECT().GetTicker(mock.Anything, mock.Anything).Return(&luno.GetTickerResponse{Pair: "XBTZAR"}, nil)
				mockClient.EXPECT().GetOrderBook(mock.Anything, mock.Anything).Return(nil, errors.New("API error"))
			},
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Unable to create order: Failed to retrieve market information for pair XBTZAR",
		},
		{
			name: "no pair for create order",
			requestParams: map[string]any{
				"type":   "BUY",
				"volume": "0.01",
				"price":  "1000000",
			},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "required argument \"pair\" not found",
		},
		{
			name: "invalid volume for create order",
			requestParams: map[string]any{
				"pair":   "XBTZAR",
				"type":   "BUY",
				"volume": "invalid_volume",
				"price":  "1000000",
			},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) { /* No mock setup needed */ },
			isAuthenticated: true,
			expectedError:   true,
			errorContains:   "Invalid volume format",
		},
		{
			name:            "unauthenticated create order",
			requestParams:   map[string]any{"pair": "XBTZAR", "type": "BUY", "volume": "0.01", "price": "1000000"},
			mockSetup:       func(t *testing.T, mockClient *sdk.MockLunoClient) {},
			isAuthenticated: false,
			expectedError:   true,
			errorContains:   ErrAPICredentialsRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := sdk.NewMockLunoClient(t)
			tt.mockSetup(t, mockClient)

			cfg := &config.Config{
				LunoClient:      mockClient,
				IsAuthenticated: tt.isAuthenticated,
			}

			handler := HandleCreateOrder(cfg)
			request := createMockRequest(tt.requestParams)
			result, err := handler(context.Background(), request)

			assert.NoError(t, err)
			if tt.expectedError {
				assert.True(t, result.IsError)
				if tt.errorContains != "" {
					errorMsg := getTextContentFromResult(t, result)
					assert.Contains(t, errorMsg, tt.errorContains)
				}
			} else {
				assert.False(t, result.IsError)
				textContent := getTextContentFromResult(t, result)
				assert.NotEmpty(t, textContent)
				assert.Contains(t, textContent, "Order created successfully!")
				assert.Contains(t, textContent, "BXMC2SEAS4KF5S2")
			}
		})
	}
}
