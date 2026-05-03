// Package info mirrors hyperliquid-python-sdk/hyperliquid/info.py.
package info

import (
	"context"
	"time"

	"hyperliquid-bot/sdk/api"
	"hyperliquid-bot/sdk/types"
	ws "hyperliquid-bot/sdk/websocket"
)

// Info corresponds to Python:
// hyperliquid.info.Info
type Info struct {
	*api.API
	CoinToAsset       map[string]int
	NameToCoin        map[string]string
	AssetToSzDecimals map[int]int
	WSManager         *ws.WebsocketManager
}

// New corresponds to Python:
// hyperliquid.info.Info.__init__
func New(baseURL string, timeout time.Duration) *Info {
	return &Info{
		API:               api.New(baseURL, timeout),
		CoinToAsset:       map[string]int{},
		NameToCoin:        map[string]string{},
		AssetToSzDecimals: map[int]int{},
	}
}

// NewInitialized corresponds to Python:
// hyperliquid.info.Info.__init__
func NewInitialized(ctx context.Context, baseURL string, skipWS bool, meta *types.Meta, spotMeta *types.SpotMeta, perpDexs []string, timeout time.Duration) (*Info, error) {
	i := New(baseURL, timeout)
	if !skipWS {
		i.WSManager = ws.NewWebsocketManager(i.BaseURL)
		if err := i.WSManager.Start(ctx); err != nil {
			return nil, err
		}
	}
	if err := i.Initialize(ctx, meta, spotMeta, perpDexs); err != nil {
		return nil, err
	}
	return i, nil
}

// Initialize corresponds to the metadata initialization part of Python:
// hyperliquid.info.Info.__init__
func (i *Info) Initialize(ctx context.Context, meta *types.Meta, spotMeta *types.SpotMeta, perpDexs []string) error {
	if spotMeta == nil {
		fresh, err := i.SpotMeta(ctx)
		if err != nil {
			return err
		}
		spotMeta = &fresh
	}
	i.setSpotMeta(*spotMeta)

	perpDexToOffset := map[string]int{"": 0}
	if perpDexs == nil {
		perpDexs = []string{""}
	} else {
		var dexs []map[string]any
		if err := i.PerpDexs(ctx, &dexs); err != nil {
			return err
		}
		for idx, perpDex := range dexs[1:] {
			name, _ := perpDex["name"].(string)
			perpDexToOffset[name] = 110000 + idx*10000
		}
	}
	for _, perpDex := range perpDexs {
		offset := perpDexToOffset[perpDex]
		if perpDex == "" && meta != nil {
			i.SetPerpMeta(*meta, 0)
			continue
		}
		fresh, err := i.Meta(ctx, perpDex)
		if err != nil {
			return err
		}
		i.SetPerpMeta(fresh, offset)
	}
	return nil
}

func (i *Info) setSpotMeta(spotMeta types.SpotMeta) {
	tokenByIndex := map[int]types.SpotTokenInfo{}
	for _, token := range spotMeta.Tokens {
		tokenByIndex[token.Index] = token
	}
	for _, spotInfo := range spotMeta.Universe {
		asset := spotInfo.Index + 10000
		i.CoinToAsset[spotInfo.Name] = asset
		i.NameToCoin[spotInfo.Name] = spotInfo.Name
		if len(spotInfo.Tokens) < 2 {
			continue
		}
		baseInfo := tokenByIndex[spotInfo.Tokens[0]]
		quoteInfo := tokenByIndex[spotInfo.Tokens[1]]
		i.AssetToSzDecimals[asset] = baseInfo.SzDecimals
		name := baseInfo.Name + "/" + quoteInfo.Name
		if _, ok := i.NameToCoin[name]; !ok {
			i.NameToCoin[name] = spotInfo.Name
		}
	}
}

// SetPerpMeta corresponds to Python:
// hyperliquid.info.Info.set_perp_meta
func (i *Info) SetPerpMeta(meta types.Meta, offset int) {
	for asset, assetInfo := range meta.Universe {
		asset += offset
		i.CoinToAsset[assetInfo.Name] = asset
		i.NameToCoin[assetInfo.Name] = assetInfo.Name
		i.AssetToSzDecimals[asset] = assetInfo.SzDecimals
	}
}

// DisconnectWebsocket corresponds to Python:
// hyperliquid.info.Info.disconnect_websocket
func (i *Info) DisconnectWebsocket() error {
	if i.WSManager == nil {
		return nil
	}
	return i.WSManager.Stop()
}

// UserState corresponds to Python:
// hyperliquid.info.Info.user_state
func (i *Info) UserState(ctx context.Context, address string, dex string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "clearinghouseState", "user": address, "dex": dex}, out)
}

// SpotUserState corresponds to Python:
// hyperliquid.info.Info.spot_user_state
func (i *Info) SpotUserState(ctx context.Context, address string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "spotClearinghouseState", "user": address}, out)
}

// OpenOrders corresponds to Python:
// hyperliquid.info.Info.open_orders
func (i *Info) OpenOrders(ctx context.Context, address string, dex string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "openOrders", "user": address, "dex": dex}, out)
}

// FrontendOpenOrders corresponds to Python:
// hyperliquid.info.Info.frontend_open_orders
func (i *Info) FrontendOpenOrders(ctx context.Context, address string, dex string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "frontendOpenOrders", "user": address, "dex": dex}, out)
}

// AllMids corresponds to Python:
// hyperliquid.info.Info.all_mids
func (i *Info) AllMids(ctx context.Context, dex string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "allMids", "dex": dex}, out)
}

// UserFills corresponds to Python:
// hyperliquid.info.Info.user_fills
func (i *Info) UserFills(ctx context.Context, address string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userFills", "user": address}, out)
}

// UserFillsByTime corresponds to Python:
// hyperliquid.info.Info.user_fills_by_time
func (i *Info) UserFillsByTime(ctx context.Context, address string, startTime int64, endTime *int64, aggregateByTime bool, out any) error {
	return i.Post(ctx, "/info", map[string]any{
		"type":            "userFillsByTime",
		"user":            address,
		"startTime":       startTime,
		"endTime":         endTime,
		"aggregateByTime": aggregateByTime,
	}, out)
}

// Meta corresponds to Python:
// hyperliquid.info.Info.meta
func (i *Info) Meta(ctx context.Context, dex string) (types.Meta, error) {
	var out types.Meta
	err := i.Post(ctx, "/info", map[string]any{"type": "meta", "dex": dex}, &out)
	return out, err
}

// SpotMeta corresponds to Python:
// hyperliquid.info.Info.spot_meta
func (i *Info) SpotMeta(ctx context.Context) (types.SpotMeta, error) {
	var out types.SpotMeta
	err := i.Post(ctx, "/info", map[string]any{"type": "spotMeta"}, &out)
	return out, err
}

// MetaAndAssetCtxs corresponds to Python:
// hyperliquid.info.Info.meta_and_asset_ctxs
func (i *Info) MetaAndAssetCtxs(ctx context.Context, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "metaAndAssetCtxs"}, out)
}

// PerpDexs corresponds to Python:
// hyperliquid.info.Info.perp_dexs
func (i *Info) PerpDexs(ctx context.Context, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "perpDexs"}, out)
}

// SpotMetaAndAssetCtxs corresponds to Python:
// hyperliquid.info.Info.spot_meta_and_asset_ctxs
func (i *Info) SpotMetaAndAssetCtxs(ctx context.Context, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "spotMetaAndAssetCtxs"}, out)
}

// FundingHistory corresponds to Python:
// hyperliquid.info.Info.funding_history
func (i *Info) FundingHistory(ctx context.Context, name string, startTime int64, endTime *int64, out any) error {
	payload := map[string]any{"type": "fundingHistory", "coin": i.NameToCoin[name], "startTime": startTime}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	return i.Post(ctx, "/info", payload, out)
}

// UserFundingHistory corresponds to Python:
// hyperliquid.info.Info.user_funding_history
func (i *Info) UserFundingHistory(ctx context.Context, user string, startTime int64, endTime *int64, out any) error {
	payload := map[string]any{"type": "userFunding", "user": user, "startTime": startTime}
	if endTime != nil {
		payload["endTime"] = *endTime
	}
	return i.Post(ctx, "/info", payload, out)
}

// L2Snapshot corresponds to Python:
// hyperliquid.info.Info.l2_snapshot
func (i *Info) L2Snapshot(ctx context.Context, name string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "l2Book", "coin": i.NameToCoin[name]}, out)
}

// CandlesSnapshot corresponds to Python:
// hyperliquid.info.Info.candles_snapshot
func (i *Info) CandlesSnapshot(ctx context.Context, name string, interval string, startTime int64, endTime int64, out any) error {
	req := map[string]any{"coin": i.NameToCoin[name], "interval": interval, "startTime": startTime, "endTime": endTime}
	return i.Post(ctx, "/info", map[string]any{"type": "candleSnapshot", "req": req}, out)
}

// UserFees corresponds to Python:
// hyperliquid.info.Info.user_fees
func (i *Info) UserFees(ctx context.Context, address string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userFees", "user": address}, out)
}

// UserStakingSummary corresponds to Python:
// hyperliquid.info.Info.user_staking_summary
func (i *Info) UserStakingSummary(ctx context.Context, address string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "delegatorSummary", "user": address}, out)
}

// UserStakingDelegations corresponds to Python:
// hyperliquid.info.Info.user_staking_delegations
func (i *Info) UserStakingDelegations(ctx context.Context, address string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "delegations", "user": address}, out)
}

// UserStakingRewards corresponds to Python:
// hyperliquid.info.Info.user_staking_rewards
func (i *Info) UserStakingRewards(ctx context.Context, address string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "delegatorRewards", "user": address}, out)
}

// DelegatorHistory corresponds to Python:
// hyperliquid.info.Info.delegator_history
func (i *Info) DelegatorHistory(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "delegatorHistory", "user": user}, out)
}

// QueryOrderByOID corresponds to Python:
// hyperliquid.info.Info.query_order_by_oid
func (i *Info) QueryOrderByOID(ctx context.Context, user string, oid int, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "orderStatus", "user": user, "oid": oid}, out)
}

// QueryOrderByCloid corresponds to Python:
// hyperliquid.info.Info.query_order_by_cloid
func (i *Info) QueryOrderByCloid(ctx context.Context, user string, cloid types.Cloid, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "orderStatus", "user": user, "oid": cloid.ToRaw()}, out)
}

// QueryReferralState corresponds to Python:
// hyperliquid.info.Info.query_referral_state
func (i *Info) QueryReferralState(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "referral", "user": user}, out)
}

// QuerySubAccounts corresponds to Python:
// hyperliquid.info.Info.query_sub_accounts
func (i *Info) QuerySubAccounts(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "subAccounts", "user": user}, out)
}

// QueryUserToMultiSigSigners corresponds to Python:
// hyperliquid.info.Info.query_user_to_multi_sig_signers
func (i *Info) QueryUserToMultiSigSigners(ctx context.Context, multiSigUser string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userToMultiSigSigners", "user": multiSigUser}, out)
}

// QueryPerpDeployAuctionStatus corresponds to Python:
// hyperliquid.info.Info.query_perp_deploy_auction_status
func (i *Info) QueryPerpDeployAuctionStatus(ctx context.Context, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "perpDeployAuctionStatus"}, out)
}

// QueryUserDexAbstractionState corresponds to Python:
// hyperliquid.info.Info.query_user_dex_abstraction_state
func (i *Info) QueryUserDexAbstractionState(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userDexAbstraction", "user": user}, out)
}

// QueryUserAbstractionState corresponds to Python:
// hyperliquid.info.Info.query_user_abstraction_state
func (i *Info) QueryUserAbstractionState(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userAbstraction", "user": user}, out)
}

// HistoricalOrders corresponds to Python:
// hyperliquid.info.Info.historical_orders
func (i *Info) HistoricalOrders(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "historicalOrders", "user": user}, out)
}

// UserNonFundingLedgerUpdates corresponds to Python:
// hyperliquid.info.Info.user_non_funding_ledger_updates
func (i *Info) UserNonFundingLedgerUpdates(ctx context.Context, user string, startTime int64, endTime *int64, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userNonFundingLedgerUpdates", "user": user, "startTime": startTime, "endTime": endTime}, out)
}

// Portfolio corresponds to Python:
// hyperliquid.info.Info.portfolio
func (i *Info) Portfolio(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "portfolio", "user": user}, out)
}

// UserTwapSliceFills corresponds to Python:
// hyperliquid.info.Info.user_twap_slice_fills
func (i *Info) UserTwapSliceFills(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userTwapSliceFills", "user": user}, out)
}

// UserVaultEquities corresponds to Python:
// hyperliquid.info.Info.user_vault_equities
func (i *Info) UserVaultEquities(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userVaultEquities", "user": user}, out)
}

// UserRole corresponds to Python:
// hyperliquid.info.Info.user_role
func (i *Info) UserRole(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userRole", "user": user}, out)
}

// UserRateLimit corresponds to Python:
// hyperliquid.info.Info.user_rate_limit
func (i *Info) UserRateLimit(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "userRateLimit", "user": user}, out)
}

// QuerySpotDeployAuctionStatus corresponds to Python:
// hyperliquid.info.Info.query_spot_deploy_auction_status
func (i *Info) QuerySpotDeployAuctionStatus(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "spotDeployState", "user": user}, out)
}

// ExtraAgents corresponds to Python:
// hyperliquid.info.Info.extra_agents
func (i *Info) ExtraAgents(ctx context.Context, user string, out any) error {
	return i.Post(ctx, "/info", map[string]any{"type": "extraAgents", "user": user}, out)
}

// RemapCoinSubscription corresponds to Python:
// hyperliquid.info.Info._remap_coin_subscription
func (i *Info) RemapCoinSubscription(subscription ws.Subscription) {
	typ, _ := subscription["type"].(string)
	switch typ {
	case "l2Book", "trades", "candle", "bbo", "activeAssetCtx":
		coin, _ := subscription["coin"].(string)
		subscription["coin"] = i.NameToCoin[coin]
	}
}

// Subscribe corresponds to Python:
// hyperliquid.info.Info.subscribe
func (i *Info) Subscribe(subscription ws.Subscription, callback ws.Callback) (int, error) {
	i.RemapCoinSubscription(subscription)
	return i.WSManager.Subscribe(subscription, callback)
}

// Unsubscribe corresponds to Python:
// hyperliquid.info.Info.unsubscribe
func (i *Info) Unsubscribe(subscription ws.Subscription, subscriptionID int) (bool, error) {
	i.RemapCoinSubscription(subscription)
	return i.WSManager.Unsubscribe(subscription, subscriptionID)
}

// NameToAsset corresponds to Python:
// hyperliquid.info.Info.name_to_asset
func (i *Info) NameToAsset(name string) int {
	return i.CoinToAsset[i.NameToCoin[name]]
}
