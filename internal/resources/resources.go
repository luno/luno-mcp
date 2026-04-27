package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/luno/luno-go"
	"github.com/luno/luno-mcp/internal/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Resource URIs
const (
	WalletResourceURI       = "luno://wallets"
	TransactionsResourceURI = "luno://transactions"
	AccountTemplateURI      = "luno://accounts/{id}"
)

// NewWalletResource creates a new resource for Luno wallets
func NewWalletResource() mcp.Resource {
	return mcp.NewResource(
		WalletResourceURI,
		"Luno Wallets",
		mcp.WithResourceDescription("Returns all wallets/balances from your Luno account"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleWalletResource returns a handler for the wallet resource
func HandleWalletResource(cfg *config.Config) server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if cfg == nil {
			return nil, fmt.Errorf("configuration is nil")
		}
		if cfg.LunoClient == nil {
			return nil, fmt.Errorf("Luno client is not configured")
		}

		balances, err := cfg.LunoClient.GetBalances(ctx, &luno.GetBalancesRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get balances: %w", err)
		}

		balancesJSON, err := json.MarshalIndent(balances, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal balances: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      WalletResourceURI,
				MIMEType: "application/json",
				Text:     string(balancesJSON),
			},
		}, nil
	}
}

// NewTransactionsResource creates a new resource for Luno transactions
func NewTransactionsResource() mcp.Resource {
	return mcp.NewResource(
		TransactionsResourceURI,
		"Luno Transactions",
		mcp.WithResourceDescription("Returns recent transactions from your Luno account"),
		mcp.WithMIMEType("application/json"),
	)
}

// HandleTransactionsResource returns a handler for the transactions resource
func HandleTransactionsResource(cfg *config.Config) server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if cfg == nil {
			return nil, fmt.Errorf("configuration is nil")
		}
		if cfg.LunoClient == nil {
			return nil, fmt.Errorf("Luno client is not configured")
		}

		// Get transactions for the first account that has them
		balances, err := cfg.LunoClient.GetBalances(ctx, &luno.GetBalancesRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get balances: %w", err)
		}

		if len(balances.Balance) == 0 {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      TransactionsResourceURI,
					MIMEType: "application/json",
					Text:     "[]",
				},
			}, nil
		}

		// Take the first account with non-zero balance
		var accountID string
		for _, balance := range balances.Balance {
			if balance.Balance.Sign() != 0 {
				accountID = balance.AccountId
				break
			}
		}

		// If no account with non-zero balance is found, use the first one
		if accountID == "" && len(balances.Balance) > 0 {
			accountID = balances.Balance[0].AccountId
		}
		// Get transactions for the selected account
		accountIDInt, err := strconv.ParseInt(accountID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse account ID: %w", err)
		}

		txnReq := &luno.ListTransactionsRequest{
			Id:     accountIDInt,
			MinRow: 0,  // Start from the first transaction
			MaxRow: 20, // Get up to 20 transactions
		}

		transactions, err := cfg.LunoClient.ListTransactions(ctx, txnReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions: %w", err)
		}

		transactionsJSON, err := json.MarshalIndent(transactions, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal transactions: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      TransactionsResourceURI,
				MIMEType: "application/json",
				Text:     string(transactionsJSON),
			},
		}, nil
	}
}

// NewAccountTemplate creates a new resource template for Luno accounts
func NewAccountTemplate() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		AccountTemplateURI,
		"Luno Account",
		mcp.WithTemplateDescription("Returns details for a specific Luno account"),
	)
}

// HandleAccountTemplate returns a handler for the account resource template
func HandleAccountTemplate(cfg *config.Config) server.ResourceTemplateHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		if cfg == nil {
			return nil, fmt.Errorf("configuration is nil")
		}
		if cfg.LunoClient == nil {
			return nil, fmt.Errorf("Luno client is not configured")
		}

		// Extract account ID from URI
		uri := request.Params.URI
		if uri == "" {
			return nil, fmt.Errorf("account ID not provided")
		}

		// Extract account ID from URI
		accountID := extractAccountID(uri)
		if accountID == "" {
			return nil, fmt.Errorf("invalid account URI format")
		}

		// Get account details
		accountReq := &luno.GetBalancesRequest{}
		balances, err := cfg.LunoClient.GetBalances(ctx, accountReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get account details: %w", err)
		}

		// Find the requested account
		var account *luno.AccountBalance
		for _, bal := range balances.Balance {
			if bal.AccountId == accountID {
				account = &bal
				break
			}
		}
		accountIDInt, err := strconv.ParseInt(accountID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse account ID: %w", err)
		}

		// Add transaction history
		txnReq := &luno.ListTransactionsRequest{
			Id:     accountIDInt,
			MinRow: 0,  // Start from the first transaction
			MaxRow: 10, // Get up to 10 transactions
		}

		transactions, err := cfg.LunoClient.ListTransactions(ctx, txnReq)
		if err != nil {
			return nil, fmt.Errorf("failed to get transactions: %w", err)
		}

		// Create a combined result with account details and transactions
		result := map[string]any{
			"account":      account,
			"transactions": transactions.Transactions,
		}

		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal account details: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(resultJSON),
			},
		}, nil
	}
}

// extractAccountID extracts the account ID from a URI like "luno://accounts/{id}"
func extractAccountID(uri string) string {
	// Simple extraction assuming the URI is in the format "luno://accounts/123"
	parts := strings.Split(uri, "/")
	// Check for invalid URI formats:
	// - len(parts) < 3: handles URIs that are too short (e.g., "luno://" or "luno:")
	// - len(parts) == 3 && parts[2] == "accounts": handles "luno://accounts" without an ID
	//   This prevents returning "accounts" as the account ID when no actual ID is provided
	if len(parts) < 3 || (len(parts) == 3 && parts[2] == "accounts") {
		return ""
	}
	return parts[len(parts)-1]
}
