package sdk

import (
	"context"

	"github.com/luno/luno-go"
)

// compile-time check that *luno.Client implements our interface
var _ LunoClient = (*luno.Client)(nil)

// LunoClient defines the interface for Luno API operations
// This interface allows us to mock the Luno client for testing
type LunoClient interface {
	GetBalances(ctx context.Context, req *luno.GetBalancesRequest) (*luno.GetBalancesResponse, error)
	GetTicker(ctx context.Context, req *luno.GetTickerRequest) (*luno.GetTickerResponse, error)
	GetOrderBook(ctx context.Context, req *luno.GetOrderBookRequest) (*luno.GetOrderBookResponse, error)
	PostLimitOrder(ctx context.Context, req *luno.PostLimitOrderRequest) (*luno.PostLimitOrderResponse, error)
	StopOrder(ctx context.Context, req *luno.StopOrderRequest) (*luno.StopOrderResponse, error)
	ListOrders(ctx context.Context, req *luno.ListOrdersRequest) (*luno.ListOrdersResponse, error)
	ListTransactions(ctx context.Context, req *luno.ListTransactionsRequest) (*luno.ListTransactionsResponse, error)
	ListTrades(ctx context.Context, req *luno.ListTradesRequest) (*luno.ListTradesResponse, error)
	GetCandles(ctx context.Context, req *luno.GetCandlesRequest) (*luno.GetCandlesResponse, error)
	GetTickers(ctx context.Context, req *luno.GetTickersRequest) (*luno.GetTickersResponse, error)
	GetOrderBookFull(ctx context.Context, req *luno.GetOrderBookFullRequest) (*luno.GetOrderBookFullResponse, error)
	Markets(ctx context.Context, req *luno.MarketsRequest) (*luno.MarketsResponse, error)
	SetBaseURL(url string)
	SetAuth(id, secret string) error
	SetDebug(debug bool)
}
