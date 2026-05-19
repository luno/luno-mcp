package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/luno/luno-go"
	"github.com/luno/luno-go/decimal"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Error messages
const (
	ErrAPICredentialsRequired = "API credentials are required for this operation. Please set LUNO_API_KEY_ID and LUNO_API_SECRET environment variables."
	ErrWriteOperationDisabled = "Write operations are disabled. To enable, restart the server with the --allow-write-operations flag or set the ALLOW_WRITE_OPERATIONS=true environment variable."
	ErrTradingPairRequired    = "Trading pair is required"
	ErrTradingPairDesc        = "Trading pair using Luno's symbol convention (e.g. XBTZAR for BTC/ZAR, ETHZAR, XBTEUR). BTC is automatically rewritten to XBT."
	AccountIDDesc             = "Numeric account ID, as a string (from get_balances)."

	writeOperationNotice = " Write operation: requires the --allow-write-operations flag or ALLOW_WRITE_OPERATIONS=true."
)

// EnhancedBalance is the per-account balance payload returned by get_balances.
// It augments the raw Luno API balance with the human-friendly account name.
type EnhancedBalance struct {
	AccountID   string `json:"account_id"`
	Asset       string `json:"asset"`
	Balance     string `json:"balance"`
	Reserved    string `json:"reserved"`
	Unconfirmed string `json:"unconfirmed"`
	Name        string `json:"name"`
}

// GetBalancesOutput is the structured response shape for get_balances.
type GetBalancesOutput struct {
	Balances []EnhancedBalance `json:"balances"`
}

// readOnlyTool composes the annotation + output-schema options shared by every
// safe read tool: title, read-only=true, destructive=false, idempotent=true,
// open-world=true, plus a generated output schema for T.
func readOnlyTool[T any](title string) mcp.ToolOption {
	return func(t *mcp.Tool) {
		for _, opt := range []mcp.ToolOption{
			mcp.WithTitleAnnotation(title),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
			withOutputSchema[T](),
		} {
			opt(t)
		}
	}
}

// structuredJSONResult marshals v to indented JSON for the human-readable text
// channel and returns it alongside v as structured content. `what` names the
// payload for the error path (e.g. "balances", "ticker").
func structuredJSONResult(v any, what string) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal %s: %v", what, err)), nil
	}
	return mcp.NewToolResultStructured(v, string(b)), nil
}

// Tool IDs
const (
	GetBalancesToolID      = "get_balances"
	GetTickerToolID        = "get_ticker"
	GetTickersToolID       = "get_tickers"
	GetOrderBookToolID     = "get_order_book"
	CreateOrderToolID      = "create_order"
	CancelOrderToolID      = "cancel_order"
	ListOrdersToolID       = "list_orders"
	ListTransactionsToolID = "list_transactions"
	GetTransactionToolID   = "get_transaction"
	ListTradesToolID       = "list_trades"
	GetCandlesToolID       = "get_candles"
	GetMarketsInfoToolID   = "get_markets_info"
	ConvertToolID          = "convert"
)

// ===== Balance Tools =====

// NewGetBalancesTool creates a new tool for getting account balances
func NewGetBalancesTool() mcp.Tool {
	return mcp.NewTool(
		GetBalancesToolID,
		mcp.WithDescription("Return balances for every account on the authenticated Luno profile, including available, reserved, and unconfirmed amounts per asset. Requires API credentials. Use this to check holdings before placing orders, conversions, or transfers; not for public market data (use get_ticker or get_tickers)."),
		readOnlyTool[GetBalancesOutput]("Get account balances"),
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

		return structuredJSONResult(GetBalancesOutput{Balances: enhancedBalances}, "balances")
	}
}

// ===== Market Tools =====

// NewGetTickerTool creates a new tool for getting ticker information
func NewGetTickerTool() mcp.Tool {
	return mcp.NewTool(
		GetTickerToolID,
		mcp.WithDescription("Get the latest ticker (last trade price, best bid, best ask, 24h rolling volume) for a single Luno trading pair. Public endpoint, no auth required. Use this for a quick price snapshot of one market; use get_tickers for multiple markets at once, or get_order_book for depth-of-book."),
		readOnlyTool[luno.GetTickerResponse]("Get ticker"),
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

		return structuredJSONResult(ticker, "ticker")
	}
}

// NewGetOrderBookTool creates a new tool for getting the order book
func NewGetOrderBookTool() mcp.Tool {
	return mcp.NewTool(
		GetOrderBookToolID,
		mcp.WithDescription("Return the top 100 bids and asks for a trading pair, aggregated by price level. Public endpoint, no auth required. Use this to assess available liquidity and likely slippage before sizing an order; use get_ticker for a simple price quote or list_trades for recent execution flow."),
		readOnlyTool[luno.GetOrderBookResponse]("Get order book"),
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

		return structuredJSONResult(orderBook, "order book")
	}
}

// NewGetTickersTool creates a new tool for getting ticker information for all currency pairs
func NewGetTickersTool() mcp.Tool {
	return mcp.NewTool(
		GetTickersToolID,
		mcp.WithDescription("List the latest tickers for all Luno trading pairs, or for a comma-separated subset. Each ticker has last trade price, best bid, best ask, and 24h volume. Public endpoint, no auth required. Use this to survey or compare multiple markets in one call; use get_ticker when you only need a single pair."),
		readOnlyTool[luno.GetTickersResponse]("List tickers"),
		mcp.WithString(
			"pair",
			mcp.Description("Optional comma-separated list of pairs to filter to (e.g. \"XBTZAR,ETHZAR\"). Omit to return every supported pair."),
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

		return structuredJSONResult(tickers, "tickers")
	}
}

// NewGetCandlesTool creates a new tool for getting candlestick market data
func NewGetCandlesTool() mcp.Tool {
	return mcp.NewTool(
		GetCandlesToolID,
		mcp.WithDescription("Return OHLCV candlestick data for a trading pair. Each candle contains timestamp (ms), open, high, low, close, and base-currency volume. Public endpoint, no auth required. Use for charts, indicators, or backtesting; not for live order-book state (use get_order_book) or trade-by-trade flow (use list_trades)."),
		readOnlyTool[luno.GetCandlesResponse]("Get candlestick data"),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description(ErrTradingPairDesc),
		),
		mcp.WithNumber(
			"since",
			mcp.Description("Start of the window as Unix epoch milliseconds (UTC). Defaults to 24 hours ago when omitted or 0."),
			mcp.Min(0),
		),
		mcp.WithNumber(
			"duration",
			mcp.Required(),
			mcp.Description("Candle duration in seconds. Common values: 60 (1m), 300 (5m), 900 (15m), 1800 (30m), 3600 (1h), 10800 (3h), 14400 (4h), 28800 (8h), 86400 (1d), 259200 (3d), 604800 (1w). The Luno API only accepts a fixed set of durations; passing an unsupported value will return an error."),
			mcp.Min(60),
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

		return structuredJSONResult(candles, "candles")
	}
}

// NewGetMarketsInfoTool creates a new tool for getting market information
func NewGetMarketsInfoTool() mcp.Tool {
	return mcp.NewTool(
		GetMarketsInfoToolID,
		mcp.WithDescription("List supported Luno markets and their trading parameters: min/max base and counter volume, price/volume tick size, fee tiers, status, and base/counter currency. Public endpoint, no auth required. Use this before placing an order to validate price/volume against the market's tick and minimum constraints; not for live prices (use get_ticker)."),
		readOnlyTool[luno.MarketsResponse]("List market info"),
		mcp.WithString(
			"pair",
			mcp.Description("Optional comma-separated list of pairs to filter to (e.g. \"XBTZAR,ETHZAR\"). Omit to return every supported market."),
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

		return structuredJSONResult(markets, "markets info")
	}
}

// ===== Trading Tools =====

// The handler always responds with an MCP tool error containing ErrWriteOperationDisabled.
func HandleWriteOperationDisabled() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError(ErrWriteOperationDisabled), nil
	}
}

// NewCreateOrderTool creates a tool that posts a new GTC limit order on Luno.
func NewCreateOrderTool() mcp.Tool {
	return mcp.NewTool(
		CreateOrderToolID,
		mcp.WithDescription("Place a new GTC limit order on Luno. The order rests on the book and may fill partially, fully, or not at all; this tool does not wait for or report fills - use list_orders to inspect state, or cancel_order to withdraw a working order. Not idempotent: repeated calls create duplicate orders. Prefer get_markets_info first to validate the pair's tick size and minimum volume."+writeOperationNotice),
		mcp.WithTitleAnnotation("Create limit order"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		withOutputSchema[luno.PostLimitOrderResponse](),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description(ErrTradingPairDesc),
		),
		mcp.WithString(
			"type",
			mcp.Required(),
			mcp.Description("Side of the order. BUY = bid (buy the base currency with the counter currency); SELL = ask (sell the base currency for the counter currency)."),
			mcp.Enum("BUY", "SELL"),
		),
		mcp.WithString(
			"volume",
			mcp.Required(),
			mcp.Description("Order volume in the base currency, as a decimal string (e.g. \"0.001\"). Must meet the market's minimum volume and step size; check get_markets_info."),
		),
		mcp.WithString(
			"price",
			mcp.Required(),
			mcp.Description("Limit price in the counter currency, as a decimal string (e.g. \"500000\"). Must align with the market's tick size; check get_markets_info."),
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
		return mcp.NewToolResultStructured(order, successMsg), nil
	}
}

// NewCancelOrderTool creates an MCP tool that cancels an existing order.
// The tool requires an "order_id" string parameter and its description indicates it is a write operation.
func NewCancelOrderTool() mcp.Tool {
	return mcp.NewTool(
		CancelOrderToolID,
		mcp.WithDescription("Cancel a working order by ID. Idempotent at the Luno API level: cancelling an already-cancelled or fully-filled order returns the current state without error. Use after list_orders to remove a specific working order."+writeOperationNotice),
		mcp.WithTitleAnnotation("Cancel order"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		withOutputSchema[luno.StopOrderResponse](),
		mcp.WithString(
			"order_id",
			mcp.Required(),
			mcp.Description("ID of the order to cancel, as returned by list_orders or create_order."),
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

		return structuredJSONResult(result, "cancel order result")
	}
}

// NewConvertTool creates a new tool for converting between currencies
func NewConvertTool() mcp.Tool {
	return mcp.NewTool(
		ConvertToolID,
		mcp.WithDescription("Instantly convert funds between two currency accounts on the authenticated profile via the Luno broker (e.g. ZAR to ZARU). The conversion is final and applies the broker quote at execution time. Pass idempotency_key to make retries safe; omitting it generates a fresh UUID per call, so retries on network error may double-convert."+writeOperationNotice),
		mcp.WithTitleAnnotation("Convert between currencies"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		withOutputSchema[luno.ConvertResponse](),
		mcp.WithString(
			"source_account_id",
			mcp.Required(),
			mcp.Description("Numeric ID of the account to debit, as a string (from get_balances)."),
		),
		mcp.WithString(
			"target_account_id",
			mcp.Required(),
			mcp.Description("Numeric ID of the account to credit, as a string (from get_balances). Must hold the target currency."),
		),
		mcp.WithString(
			"amount",
			mcp.Required(),
			mcp.Description("Amount of the source currency to convert, as a decimal string (e.g. \"100.00\"). Must be positive."),
		),
		mcp.WithString(
			"idempotency_key",
			mcp.Description("Any unique string up to 255 chars. Reuse the same key to safely retry a failed call without double-converting. Auto-generated as a UUID when omitted."),
			mcp.MaxLength(255),
		),
	)
}

// HandleConvert handles the convert tool
func HandleConvert(cfg *config.Config) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !cfg.IsAuthenticated {
			return mcp.NewToolResultError(ErrAPICredentialsRequired), nil
		}

		sourceAccountIDStr, err := request.RequireString("source_account_id")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting source_account_id from request", err), nil
		}
		sourceAccountID, err := strconv.ParseInt(sourceAccountIDStr, 10, 64)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid source account ID format: %v", err)), nil
		}
		if sourceAccountID <= 0 {
			return mcp.NewToolResultError("Invalid source account ID: must be a positive integer"), nil
		}

		targetAccountIDStr, err := request.RequireString("target_account_id")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting target_account_id from request", err), nil
		}
		targetAccountID, err := strconv.ParseInt(targetAccountIDStr, 10, 64)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid target account ID format: %v", err)), nil
		}
		if targetAccountID <= 0 {
			return mcp.NewToolResultError("Invalid target account ID: must be a positive integer"), nil
		}

		amountStr, err := request.RequireString("amount")
		if err != nil {
			return mcp.NewToolResultErrorFromErr("getting amount from request", err), nil
		}
		amount, err := decimal.NewFromString(amountStr)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid amount format: %v", err)), nil
		}
		if amount.Sign() <= 0 {
			return mcp.NewToolResultError("amount must be positive"), nil
		}

		idempotencyKey := request.GetString("idempotency_key", "")
		if idempotencyKey == "" {
			uid, err := uuid.NewRandom()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to generate idempotency key: %v", err)), nil
			}
			idempotencyKey = uid.String()
		}

		result, err := cfg.LunoClient.Convert(ctx, &luno.ConvertRequest{
			SourceAccountId: sourceAccountID,
			TargetAccountId: targetAccountID,
			Amount:          amount,
			IdempotencyKey:  idempotencyKey,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to convert: %v", err)), nil
		}

		return structuredJSONResult(result, "convert result")
	}
}

// NewListOrdersTool creates a new tool for listing orders
func NewListOrdersTool() mcp.Tool {
	return mcp.NewTool(
		ListOrdersToolID,
		mcp.WithDescription("List orders on the authenticated profile (open by default; recently completed orders may also appear depending on Luno API behaviour), optionally filtered by trading pair. Requires API credentials. Use this to inspect or audit working orders before cancelling them or placing new ones."),
		readOnlyTool[luno.ListOrdersResponse]("List open orders"),
		mcp.WithString(
			"pair",
			mcp.Description("Optional trading pair filter (e.g. \"XBTZAR\"). Omit to list orders across all pairs."),
		),
		mcp.WithNumber(
			"limit",
			mcp.Description("Maximum number of orders to return (1-100). Defaults to 100."),
			mcp.Min(1),
			mcp.Max(100),
			mcp.DefaultNumber(100),
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
		if pair != "" {
			pair = normalizeCurrencyPair(pair)
		}

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

		return structuredJSONResult(orders, "orders")
	}
}

// ===== Transaction Tools =====

// NewListTransactionsTool creates a new tool for listing transactions
func NewListTransactionsTool() mcp.Tool {
	return mcp.NewTool(
		ListTransactionsToolID,
		mcp.WithDescription("List ledger entries for a single account on the authenticated profile, paginated by row ID. min_row is inclusive, max_row is exclusive, and the API caps a single window at 1000 rows. Requires API credentials. Use for reconciliation, exports, or audit trails; iterate by advancing min_row to (last returned row + 1)."),
		readOnlyTool[luno.ListTransactionsResponse]("List account transactions"),
		mcp.WithString(
			"account_id",
			mcp.Required(),
			mcp.Description(AccountIDDesc),
		),
		mcp.WithNumber(
			"min_row",
			mcp.Description("First row to return, inclusive (1-based). Defaults to 1 when omitted."),
			mcp.Min(1),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber(
			"max_row",
			mcp.Description("Last row to return, exclusive. Defaults to 100 when omitted. The Luno API caps (max_row - min_row) at 1000."),
			mcp.Min(1),
			mcp.DefaultNumber(100),
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

		return structuredJSONResult(transactions, "transactions")
	}
}

// NewGetTransactionTool creates a new tool for getting a specific transaction
func NewGetTransactionTool() mcp.Tool {
	return mcp.NewTool(
		GetTransactionToolID,
		mcp.WithDescription("Return the full ledger entry for one transaction on the authenticated profile (amount, running balance, description, timestamp). Only finds transactions within the most recent 1000 rows of the account; for older entries, narrow the search with list_transactions first. Requires API credentials."),
		readOnlyTool[luno.Transaction]("Get transaction details"),
		mcp.WithString(
			"account_id",
			mcp.Required(),
			mcp.Description(AccountIDDesc),
		),
		mcp.WithString(
			"transaction_id",
			mcp.Required(),
			mcp.Description("Row index of the transaction within the account, as a string. Obtain from list_transactions (row_index field)."),
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

		return structuredJSONResult(transaction, "transaction")
	}
}

// ===== Trades Tools =====

// NewListTradesTool creates a new tool for listing trades
func NewListTradesTool() mcp.Tool {
	return mcp.NewTool(
		ListTradesToolID,
		mcp.WithDescription("List recent public trades for a trading pair, with price, volume, side (BUY/SELL), and timestamp. Public endpoint, no auth required. Use for recent execution flow or tape analysis; not for your own trade history."),
		readOnlyTool[luno.ListTradesResponse]("List recent trades"),
		mcp.WithString(
			"pair",
			mcp.Required(),
			mcp.Description(ErrTradingPairDesc),
		),
		mcp.WithString(
			"since",
			mcp.Description("Optional lower bound as Unix epoch milliseconds (UTC), passed as a string. Only trades executed after this time are returned. Omit for the most recent trades."),
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

		return structuredJSONResult(trades, "trades")
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
