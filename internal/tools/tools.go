package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/luno/luno-go"
	"github.com/luno/luno-go/decimal"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Error messages
const (
	ErrAPICredentialsRequired = "API credentials are required for this operation. Please set LUNO_API_KEY_ID and LUNO_API_SECRET environment variables."
	ErrTradingPairRequired    = "Trading pair is required"
	ErrTradingPairDesc        = "Trading pair (e.g., XBTZAR)"
)

// Tool IDs
const (
	GetBalancesToolID      = "get_balances"
	GetTickerToolID        = "get_ticker"
	GetOrderBookToolID     = "get_order_book"
	CreateOrderToolID      = "create_order"
	CancelOrderToolID      = "cancel_order"
	ListOrdersToolID       = "list_orders"
	ListTransactionsToolID = "list_transactions"
	GetTransactionToolID   = "get_transaction"
	ListTradesToolID       = "list_trades"
	GetTickersToolID       = "get_tickers"
	GetCandlesToolID       = "get_candles"
	GetMarketsInfoToolID   = "get_markets_info"
)

// ===== Balance Tools =====

// NewGetBalancesTool creates a new tool for getting account balances
func NewGetBalancesTool() mcp.Tool {
	return mcp.NewTool(
		GetBalancesToolID,
		mcp.WithDescription("Get balances for all Luno accounts"),
	)
}

// HandleGetBalances handles the get_balances tool
func HandleGetBalances(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !cfg.IsAuthenticated {
			return mcp.NewToolResultError(ErrAPICredentialsRequired), nil
		}

		balances, err := cfg.LunoClient.GetBalances(ctx, &luno.GetBalancesRequest{})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get balances: %v", err)), nil
		}

		// Enhance the response with additional information
		type EnhancedBalance struct {
			AccountID   string `json:"account_id"`
			Asset       string `json:"asset"`
			Balance     string `json:"balance"`
			Reserved    string `json:"reserved"`
			Unconfirmed string `json:"unconfirmed"`
			Name        string `json:"name"`
		}

		enhancedBalances := make([]EnhancedBalance, 0, len(balances.Balance))
		for _, balance := range balances.Balance {
			enhancedBalances = append(enhancedBalances, EnhancedBalance{
				AccountID:   balance.AccountId,
				Asset:       balance.Asset,
				Balance:     balance.Balance.String(),
				Reserved:    balance.Reserved.String(),
				Unconfirmed: balance.Unconfirmed.String(),
				Name:        balance.Name,
			})
		}

		resultJSON, err := json.MarshalIndent(enhancedBalances, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal balances: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// ===== Market Tools =====

// NewGetTickerTool creates a new tool for getting ticker information
func NewGetTickerTool() mcp.Tool {
	return mcp.NewTool(
		GetTickerToolID,
		mcp.WithDescription("Get ticker information for a trading pair"),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description(ErrTradingPairDesc),
		),
	)
}

// HandleGetTicker handles the get_ticker tool
func HandleGetTicker(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pair, err := request.RequireString("pair")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting pair from request", err), nil
		}

		// Normalize currency pair
		pair = normalizeCurrencyPair(pair)

		ticker, err := cfg.LunoClient.GetTicker(ctx, &luno.GetTickerRequest{
			Pair: pair,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting ticker", err), nil
		}

		resultJSON, err := json.MarshalIndent(ticker, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal ticker: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// NewGetOrderBookTool creates a new tool for getting the order book
func NewGetOrderBookTool() mcp.Tool {
	return mcp.NewTool(
		GetOrderBookToolID,
		mcp.WithDescription("Get order book for a trading pair"),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description(ErrTradingPairDesc),
		),
	)
}

// HandleGetOrderBook handles the get_order_book tool
func HandleGetOrderBook(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pair, err := request.RequireString("pair")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting pair from request", err), nil
		}

		// Normalize currency pair
		pair = normalizeCurrencyPair(pair)

		orderBook, err := cfg.LunoClient.GetOrderBook(ctx, &luno.GetOrderBookRequest{
			Pair: pair,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting order book", err), nil
		}

		resultJSON, err := json.MarshalIndent(orderBook, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal order book: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// NewGetTickersTool creates a new tool for getting ticker information for all currency pairs
func NewGetTickersTool() mcp.Tool {
	return mcp.NewTool(
		GetTickersToolID,
		mcp.WithDescription("List tickers for all currency pairs"),
		mcp.WithString(
			"pair",
			mcp.Description("Return tickers for multiple markets (e.g., XBTZAR,ETHZAR)"),
		),
	)
}

// HandleGetTickers handles the get_tickers tool
func HandleGetTickers(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pairsStr := request.GetString("pair", "")
		var pairs []string
		if pairsStr != "" {
			pairs = strings.Split(pairsStr, ",")
			for i, p := range pairs {
				pairs[i] = normalizeCurrencyPair(p)
			}
		}

		tickers, err := cfg.LunoClient.GetTickers(ctx, &luno.GetTickersRequest{
			Pair: pairs,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting tickers", err), nil
		}

		resultJSON, err := json.MarshalIndent(tickers, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal tickers: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// NewGetCandlesTool creates a new tool for getting candlestick market data
func NewGetCandlesTool() mcp.Tool {
	return mcp.NewTool(
		GetCandlesToolID,
		mcp.WithDescription("Get candlestick market data for a currency pair"),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description(ErrTradingPairDesc),
		),
		mcp.WithNumber(
			"since",
			mcp.Description("Filter to candles starting on or after this timestamp (Unix milliseconds). Defaults to 24 hours ago."),
		),
		mcp.WithNumber(
			"duration",
			mcp.Required(),
			mcp.Description("Candle duration in seconds (e.g., 60 for 1m, 300 for 5m, 3600 for 1h)"),
		),
	)
}

// HandleGetCandles handles the get_candles tool
func HandleGetCandles(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pair, err := request.RequireString("pair")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting pair from request", err), nil
		}
		pair = normalizeCurrencyPair(pair)

		sinceFloat := request.GetFloat("since", 0)
		var since luno.Time
		if sinceFloat == 0 {
			// Default to 24 hours ago if since is not provided or is 0
			since = luno.Time(time.Now().Add(-24 * time.Hour))
		} else {
			since = luno.Time(time.UnixMilli(int64(sinceFloat)))
		}

		durationFloat, err := request.RequireFloat("duration")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting duration from request", err), nil
		}
		duration := int64(durationFloat)

		candles, err := cfg.LunoClient.GetCandles(ctx, &luno.GetCandlesRequest{
			Pair:     pair,
			Since:    since,
			Duration: duration,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting candles", err), nil
		}

		resultJSON, err := json.MarshalIndent(candles, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal candles: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// NewGetMarketsInfoTool creates a new tool for getting market information
func NewGetMarketsInfoTool() mcp.Tool {
	return mcp.NewTool(
		GetMarketsInfoToolID,
		mcp.WithDescription("List all supported markets parameter information"),
		mcp.WithString(
			"pair",
			mcp.Description("List of market pairs to return (e.g., XBTZAR,ETHZAR)"),
		),
	)
}

// HandleGetMarketsInfo handles the get_markets_info tool
func HandleGetMarketsInfo(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pairsStr := request.GetString("pair", "")
		var pairs []string
		if pairsStr != "" {
			pairs = strings.Split(pairsStr, ",")
			for i, p := range pairs {
				pairs[i] = normalizeCurrencyPair(p)
			}
		}

		markets, err := cfg.LunoClient.Markets(ctx, &luno.MarketsRequest{
			Pair: pairs,
		})
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting markets info", err), nil
		}

		resultJSON, err := json.MarshalIndent(markets, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal markets info: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// ===== Trading Tools =====

// NewCreateOrderTool creates a new tool for creating limit orders
func NewCreateOrderTool() mcp.Tool {
	return mcp.NewTool(
		CreateOrderToolID,
		mcp.WithDescription("Create a new limit order"),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description("Trading pair (e.g., XBTZAR)"),
		),
		mcp.WithString(
			"type",
			mcp.Required(),
			mcp.Description("Order type (BUY or SELL)"),
			mcp.Enum("BUY", "SELL"),
		),
		mcp.WithString(
			"volume",
			mcp.Required(),
			mcp.Description("Order volume (amount of cryptocurrency to buy or sell)"),
		),
		mcp.WithString(
			"price",
			mcp.Required(),
			mcp.Description("Limit price as a decimal string"),
		),
	)
}

// HandleCreateOrder handles the create_order tool for limit orders
// TODO: Add HandleCreateMarketOrder function for market orders
func HandleCreateOrder(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !cfg.IsAuthenticated {
			return mcp.NewToolResultError(ErrAPICredentialsRequired), nil
		}

		pair, err := request.RequireString("pair")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting pair from request", err), nil
		}
		slog.Debug("Processing trading pair", "originalPair", pair)

		// Normalize the pair - this should handle BTC->XBT conversion automatically
		pair = normalizeCurrencyPair(pair)
		slog.Debug("Normalized trading pair", "originalPair", pair, "normalizedPair", pair)

		orderType, err := request.RequireString("type")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting type from request", err), nil
		}
		if orderType != "BUY" && orderType != "SELL" {
			return mcp.NewToolResultError("Order type must be 'BUY' or 'SELL'"), nil
		}

		volumeStr, err := request.RequireString("volume")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting volume from request", err), nil
		}

		priceStr, err := request.RequireString("price")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting price from request", err), nil
		}

		// Validate numeric values
		volumeDec, err := decimal.NewFromString(volumeStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid volume format: %v", err)), nil
		}

		priceDec, err := decimal.NewFromString(priceStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid price format: %v", err)), nil
		}

		// Map BUY/SELL to BID/ASK for limit orders
		var lunoOrderType luno.OrderType
		if orderType == "BUY" {
			lunoOrderType = luno.OrderTypeBid
		} else { // SELL
			lunoOrderType = luno.OrderTypeAsk
		}

		// Get market info - we already validated the pair, but this provides additional info
		marketInfoString, err := GetMarketInfo(ctx, cfg, pair)
		if err != nil {
			slog.Error("Failed to get market info during order creation", "pair", pair, "error", err)
			return mcp.NewToolResultError(fmt.Sprintf("Unable to create order: Failed to retrieve market information for pair %s. Details: %v", pair, err)), nil
		}

		// Log the request parameters for debugging
		slog.Info("Creating order",
			"pair", pair,
			"type", lunoOrderType,
			"volume", volumeDec.String(),
			"price", priceDec.String())

		// Create the limit order
		createReq := &luno.PostLimitOrderRequest{
			Pair:   pair,
			Type:   lunoOrderType,
			Volume: volumeDec,
			Price:  priceDec,
		}

		order, err := cfg.LunoClient.PostLimitOrder(ctx, createReq)
		if err != nil {
			// If the order fails despite our validation, provide detailed error information
			errorMsg := fmt.Sprintf("Failed to create limit order: %v\n\n"+
				"Here's what we know about this market:\n%s\n\n"+
				"This may be due to insufficient balance, market conditions, or API limits.",
				err, marketInfoString)

			return mcp.NewToolResultError(errorMsg), nil
		}

		// Order succeeded
		resultJSON, err := json.MarshalIndent(order, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal order result: %v", err)), nil
		}

		successMsg := fmt.Sprintf("Order created successfully!\n\n%s\n\n%s",
			string(resultJSON), marketInfoString)
		return mcp.NewToolResultText(successMsg), nil
	}
}

// NewCancelOrderTool creates a new tool for canceling orders
func NewCancelOrderTool() mcp.Tool {
	return mcp.NewTool(
		CancelOrderToolID,
		mcp.WithDescription("Cancel an order"),
		mcp.WithString(
			"order_id",
			mcp.Required(),
			mcp.Description("Order ID to cancel"),
		),
	)
}

// HandleCancelOrder handles the cancel_order tool
func HandleCancelOrder(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !cfg.IsAuthenticated {
			return mcp.NewToolResultError(ErrAPICredentialsRequired), nil
		}

		orderID, err := request.RequireString("order_id")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting order_id from request", err), nil
		}

		result, err := cfg.LunoClient.StopOrder(ctx, &luno.StopOrderRequest{
			OrderId: orderID,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to cancel order: %v", err)), nil
		}

		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// NewListOrdersTool creates a new tool for listing orders
func NewListOrdersTool() mcp.Tool {
	return mcp.NewTool(
		ListOrdersToolID,
		mcp.WithDescription("List open orders"),
		mcp.WithString(
			"pair",
			mcp.Description("Trading pair (e.g., XBTZAR)"),
		),
		mcp.WithNumber(
			"limit",
			mcp.Description("Maximum number of orders to return (default: 100)"),
		),
	)
}

// HandleListOrders handles the list_orders tool
func HandleListOrders(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !cfg.IsAuthenticated {
			return mcp.NewToolResultError(ErrAPICredentialsRequired), nil
		}

		// Get the pair if provided, otherwise it will be an empty string.
		// An empty pair string will result in fetching orders for all pairs.
		pair := request.GetString("pair", "")

		// Default to 100 if not present
		limit := request.GetFloat("limit", 100)

		listReq := &luno.ListOrdersRequest{
			Pair:  pair,
			Limit: int64(limit),
		}

		orders, err := cfg.LunoClient.ListOrders(ctx, listReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list orders: %v", err)), nil
		}

		resultJSON, err := json.MarshalIndent(orders, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal orders: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// ===== Transaction Tools =====

// NewListTransactionsTool creates a new tool for listing transactions
func NewListTransactionsTool() mcp.Tool {
	return mcp.NewTool(
		ListTransactionsToolID,
		mcp.WithDescription("List transactions for an account"),
		mcp.WithString(
			"account_id",
			mcp.Required(),
			mcp.Description("Account ID"),
		),
		mcp.WithNumber(
			"min_row",
			mcp.Description("Minimum row ID to return (for pagination, inclusive)"),
		),
		mcp.WithNumber(
			"max_row",
			mcp.Description("Maximum row ID to return (for pagination, exclusive)"),
		),
	)
}

// HandleListTransactions handles the list_transactions tool
func HandleListTransactions(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !cfg.IsAuthenticated {
			return mcp.NewToolResultError(ErrAPICredentialsRequired), nil
		}

		accountIDStr, err := request.RequireString("account_id")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting account_id from request", err), nil
		}

		// Convert account ID from string to int64
		accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid account ID format: %v. Please provide a valid numeric account ID.", err)), nil
		}

		listReq := &luno.ListTransactionsRequest{
			Id: accountID,
		}

		// Default to 1 if not present
		minRow := request.GetInt("min_row", 1)
		listReq.MinRow = int64(minRow)

		// Default to 100 if not present
		maxRow := request.GetInt("max_row", 100)
		listReq.MaxRow = int64(maxRow)

		transactions, err := cfg.LunoClient.ListTransactions(ctx, listReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list transactions: %v", err)), nil
		}

		resultJSON, err := json.MarshalIndent(transactions, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal transactions: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// NewGetTransactionTool creates a new tool for getting a specific transaction
func NewGetTransactionTool() mcp.Tool {
	return mcp.NewTool(
		GetTransactionToolID,
		mcp.WithDescription("Get details of a specific transaction"),
		mcp.WithString(
			"account_id",
			mcp.Required(),
			mcp.Description("Account ID"),
		),
		mcp.WithString(
			"transaction_id",
			mcp.Required(),
			mcp.Description("Transaction ID"),
		),
	)
}

// HandleGetTransaction handles the get_transaction tool
func HandleGetTransaction(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !cfg.IsAuthenticated {
			return mcp.NewToolResultError(ErrAPICredentialsRequired), nil
		}

		accountIDStr, err := request.RequireString("account_id")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting account_id from request", err), nil
		}

		// Convert account ID from string to int64
		accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid account ID format: %v. Please provide a valid numeric account ID.", err)), nil
		}

		transactionIDStr, err := request.RequireString("transaction_id")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting transaction_id from request", err), nil
		}

		// Attempt to convert transaction ID to int64 for comparison
		transactionID, err := strconv.ParseInt(transactionIDStr, 10, 64)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid transaction ID format: %v. Please provide a valid numeric transaction ID.", err)), nil
		}

		// Get the list of transactions with MinRow and MaxRow
		listReq := &luno.ListTransactionsRequest{
			Id:     accountID,
			MinRow: 0,    // Start from the beginning
			MaxRow: 1000, // Use a reasonable max to find the transaction
		}

		transactions, err := cfg.LunoClient.ListTransactions(ctx, listReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get transactions: %v", err)), nil
		}

		// Find the specific transaction
		var transaction *luno.Transaction
		for _, txn := range transactions.Transactions {
			if txn.RowIndex == transactionID {
				transaction = &txn
				break
			}
		}

		if transaction == nil {
			return mcp.NewToolResultError(fmt.Sprintf("Transaction not found: %s", transactionIDStr)), nil
		}

		resultJSON, err := json.MarshalIndent(transaction, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal transaction: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// ===== Trades Tools =====

// NewListTradesTool creates a new tool for listing trades
func NewListTradesTool() mcp.Tool {
	return mcp.NewTool(
		ListTradesToolID,
		mcp.WithDescription("List recent trades for a currency pair"),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description(ErrTradingPairDesc),
		),
		mcp.WithString(
			"since",
			mcp.Description("Fetch trades executed after this timestamp (Unix milliseconds)"),
		),
	)
}

// HandleListTrades handles the list_trades tool
func HandleListTrades(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// This is a public endpoint, so no authentication check is needed here.
		// However, the LunoClient.ListTrades method might still require authentication
		// depending on the underlying luno-go library implementation.
		// For now, we assume it can be called unauthenticated.

		pair, err := request.RequireString("pair")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting pair from request", err), nil
		}

		// Normalize currency pair
		pair = normalizeCurrencyPair(pair)

		req := &luno.ListTradesRequest{
			Pair: pair,
		}

		sinceStr := request.GetString("since", "")
		if sinceStr != "" {
			// Try to parse the since timestamp
			sinceInt, err := strconv.ParseInt(sinceStr, 10, 64)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid 'since' timestamp format: %v. Please provide a valid Unix millisecond timestamp.", err)), nil
			}
			req.Since = luno.Time(time.UnixMilli(sinceInt))
		}

		trades, err := cfg.LunoClient.ListTrades(ctx, req)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("listing trades", err), nil
		}

		resultJSON, err := json.MarshalIndent(trades, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal trades: %v", err)), nil
		}

		return mcp.NewToolResultText(string(resultJSON)), nil
	}
}

// ===== Helper Functions =====

// normalizeCurrencyPair converts common currency pair formats to Luno's expected format
func normalizeCurrencyPair(pair string) string {
	// Log input for debugging
	originalPair := pair

	// Remove any separators that might be in the pair
	pair = strings.Replace(pair, "-", "", -1)
	pair = strings.Replace(pair, "_", "", -1)
	pair = strings.Replace(pair, "/", "", -1)
	pair = strings.ToUpper(pair)

	// Apply currency code standardization
	// Known mappings between common symbols and Luno's expected format
	currencyMappings := map[string]string{
		"BTC":     "XBT", // Bitcoin is XBT on Luno
		"BITCOIN": "XBT",
		// Add other mappings if needed in the future
	}

	// Apply all mappings
	for common, luno := range currencyMappings {
		pair = strings.Replace(pair, common, luno, -1)
	}

	// Log the normalization for debugging
	slog.Debug("Currency pair normalization", "original", originalPair, "normalized", pair)

	return pair
}
