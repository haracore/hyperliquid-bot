package client

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"hyperliquid-bot/sdk/exchange"
	hlinfo "hyperliquid-bot/sdk/info"
	"hyperliquid-bot/sdk/signing"
	"hyperliquid-bot/sdk/types"
)

type Client struct {
	baseURL      string
	timeout      time.Duration
	privateKey   string
	dex          string
	vaultAddress *string
}

type Cloid = types.Cloid
type OidOrCloid = exchange.OidOrCloid

type Config struct {
	BaseURL      string
	Timeout      time.Duration
	PrivateKey   string
	Dex          string
	VaultAddress *string
}

func NewCloid(raw string) (Cloid, error) {
	return types.NewCloid(raw)
}

func ParseOrderID(oid int, cloidRaw string) (OidOrCloid, error) {
	if cloidRaw != "" {
		return NewCloid(cloidRaw)
	}
	return oid, nil
}

func New(config Config) *Client {
	return &Client{
		baseURL:      config.BaseURL,
		timeout:      config.Timeout,
		privateKey:   config.PrivateKey,
		dex:          config.Dex,
		vaultAddress: config.VaultAddress,
	}
}

type BalancesResult struct {
	PerpState map[string]any
	SpotState map[string]any
}

func (c *Client) Balances(ctx context.Context, address string) (BalancesResult, error) {
	info := hlinfo.New(c.baseURL, c.timeout)

	var result BalancesResult
	if err := info.UserState(ctx, address, "", &result.PerpState); err != nil {
		return result, fmt.Errorf("perp state: %w", err)
	}
	if err := info.SpotUserState(ctx, address, &result.SpotState); err != nil {
		return result, fmt.Errorf("spot state: %w", err)
	}
	return result, nil
}

type Position struct {
	Coin          string
	Szi           string
	EntryPx       string
	PositionValue string
	UnrealizedPnl string
	MarginUsed    string
	Leverage      string
}

func (c *Client) PerpPositions(ctx context.Context, address string, showAll bool) ([]Position, error) {
	info := hlinfo.New(c.baseURL, c.timeout)
	var state map[string]any
	if err := info.UserState(ctx, address, c.dex, &state); err != nil {
		return nil, fmt.Errorf("user state: %w", err)
	}
	return ExtractPerpPositions(state, showAll), nil
}

func ExtractPerpPositions(state map[string]any, showAll bool) []Position {
	rawPositions, _ := state["assetPositions"].([]any)
	rows := make([]Position, 0, len(rawPositions))
	for _, raw := range rawPositions {
		wrapper, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		position, ok := wrapper["position"].(map[string]any)
		if !ok {
			continue
		}
		szi := stringValue(position["szi"])
		if !showAll && isZero(szi) {
			continue
		}
		rows = append(rows, Position{
			Coin:          stringValue(position["coin"]),
			Szi:           szi,
			EntryPx:       stringValue(position["entryPx"]),
			PositionValue: stringValue(position["positionValue"]),
			UnrealizedPnl: stringValue(position["unrealizedPnl"]),
			MarginUsed:    stringValue(position["marginUsed"]),
			Leverage:      leverageValue(position["leverage"]),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Coin < rows[j].Coin
	})
	return rows
}

func (c *Client) PerpOpenOrders(ctx context.Context, address string) (any, error) {
	info := hlinfo.New(c.baseURL, c.timeout)
	var response any
	if err := info.OpenOrders(ctx, address, c.dex, &response); err != nil {
		return nil, fmt.Errorf("perp open orders: %w", err)
	}
	return response, nil
}

func (c *Client) SpotOpenOrders(ctx context.Context, address string, frontend bool) (any, error) {
	info := hlinfo.New(c.baseURL, c.timeout)
	var response any
	var err error
	if frontend {
		err = info.FrontendOpenOrders(ctx, address, "", &response)
	} else {
		err = info.OpenOrders(ctx, address, "", &response)
	}
	if err != nil {
		return nil, fmt.Errorf("spot open orders: %w", err)
	}
	return response, nil
}

type OrderRequest struct {
	Coin       string
	IsBuy      bool
	Size       float64
	Price      float64
	TIF        string
	ReduceOnly bool
	Cloid      *Cloid
}

func (c *Client) PlacePerpOrder(ctx context.Context, request OrderRequest) (any, error) {
	return c.placeOrder(ctx, request)
}

func (c *Client) PlaceSpotOrder(ctx context.Context, request OrderRequest) (any, error) {
	request.ReduceOnly = false
	return c.placeOrder(ctx, request)
}

type CancelOrderRequest struct {
	Coin string
	Oid  int
}

func (c *Client) CancelPerpOrder(ctx context.Context, request CancelOrderRequest) (any, error) {
	return c.cancelOrder(ctx, request)
}

func (c *Client) CancelSpotOrder(ctx context.Context, request CancelOrderRequest) (any, error) {
	return c.cancelOrder(ctx, request)
}

type CancelByCloidRequest struct {
	Coin  string
	Cloid Cloid
}

func (c *Client) CancelPerpByCloid(ctx context.Context, request CancelByCloidRequest) (any, error) {
	return c.cancelByCloid(ctx, request)
}

func (c *Client) CancelSpotByCloid(ctx context.Context, request CancelByCloidRequest) (any, error) {
	return c.cancelByCloid(ctx, request)
}

type ModifyOrderRequest struct {
	OrderID    OidOrCloid
	Coin       string
	IsBuy      bool
	Size       float64
	Price      float64
	TIF        string
	ReduceOnly bool
	Cloid      *Cloid
}

func (c *Client) ModifyPerpOrder(ctx context.Context, request ModifyOrderRequest) (any, error) {
	return c.modifyOrder(ctx, request)
}

func (c *Client) ModifySpotOrder(ctx context.Context, request ModifyOrderRequest) (any, error) {
	request.ReduceOnly = false
	return c.modifyOrder(ctx, request)
}

func (c *Client) placeOrder(ctx context.Context, request OrderRequest) (any, error) {
	ex, err := c.newExchange(ctx)
	if err != nil {
		return nil, err
	}
	var response any
	err = ex.Order(
		ctx,
		request.Coin,
		request.IsBuy,
		request.Size,
		request.Price,
		limitOrderType(request.TIF),
		request.ReduceOnly,
		request.Cloid,
		nil,
		&response,
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) cancelOrder(ctx context.Context, request CancelOrderRequest) (any, error) {
	ex, err := c.newExchange(ctx)
	if err != nil {
		return nil, err
	}
	var response any
	if err := ex.Cancel(ctx, request.Coin, request.Oid, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) cancelByCloid(ctx context.Context, request CancelByCloidRequest) (any, error) {
	ex, err := c.newExchange(ctx)
	if err != nil {
		return nil, err
	}
	var response any
	if err := ex.CancelByCloid(ctx, request.Coin, request.Cloid, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) modifyOrder(ctx context.Context, request ModifyOrderRequest) (any, error) {
	ex, err := c.newExchange(ctx)
	if err != nil {
		return nil, err
	}
	var response any
	err = ex.ModifyOrder(
		ctx,
		request.OrderID,
		request.Coin,
		request.IsBuy,
		request.Size,
		request.Price,
		limitOrderType(request.TIF),
		request.ReduceOnly,
		request.Cloid,
		&response,
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) newExchange(ctx context.Context) (*exchange.Exchange, error) {
	wallet, err := signing.PrivateKeyFromHex(c.privateKey)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	perpDexs := []string{""}
	if c.dex != "" {
		perpDexs = []string{c.dex}
	}
	info, err := hlinfo.NewInitialized(ctx, c.baseURL, true, nil, nil, perpDexs, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("initialize info metadata: %w", err)
	}
	ex := exchange.New(wallet, c.baseURL, c.timeout, c.vaultAddress, nil)
	ex.Info = info
	return ex, nil
}

func limitOrderType(tif string) signing.OrderType {
	return signing.OrderType{"limit": map[string]any{"tif": tif}}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func isZero(value string) bool {
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed == 0
}

func leverageValue(value any) string {
	lev, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	typ := stringValue(lev["type"])
	raw := stringValue(lev["value"])
	if typ == "" {
		return raw
	}
	if raw == "" {
		return typ
	}
	return typ + ":" + raw
}
