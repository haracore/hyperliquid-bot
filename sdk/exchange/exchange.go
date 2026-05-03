// Package exchange mirrors hyperliquid-python-sdk/hyperliquid/exchange.py.
package exchange

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"hyperliquid-bot/sdk/api"
	"hyperliquid-bot/sdk/constants"
	"hyperliquid-bot/sdk/info"
	"hyperliquid-bot/sdk/signing"
	"hyperliquid-bot/sdk/types"

	"github.com/ethereum/go-ethereum/crypto"
)

// Exchange corresponds to Python:
// hyperliquid.exchange.Exchange
type Exchange struct {
	*api.API
	Wallet         *ecdsa.PrivateKey
	Info           *info.Info
	VaultAddress   *string
	AccountAddress *string
	ExpiresAfter   *int64
	IsMainnet      bool
}

const DefaultSlippage = 0.05

// New corresponds to Python:
// hyperliquid.exchange.Exchange.__init__
func New(wallet *ecdsa.PrivateKey, baseURL string, timeout time.Duration, vaultAddress *string, accountAddress *string) *Exchange {
	return &Exchange{
		API:            api.New(baseURL, timeout),
		Wallet:         wallet,
		Info:           info.New(baseURL, timeout),
		VaultAddress:   vaultAddress,
		AccountAddress: accountAddress,
		IsMainnet:      baseURL == "" || baseURL == constants.MainnetAPIURL,
	}
}

// GetDex corresponds to Python:
// hyperliquid.exchange._get_dex
func GetDex(coin string) string {
	if strings.Contains(coin, ":") {
		return strings.SplitN(coin, ":", 2)[0]
	}
	return ""
}

// SlippagePrice corresponds to Python:
// hyperliquid.exchange.Exchange._slippage_price
func (e *Exchange) SlippagePrice(ctx context.Context, name string, isBuy bool, slippage float64, px *float64) (float64, error) {
	coin := e.Info.NameToCoin[name]
	price := 0.0
	if px == nil {
		dex := GetDex(coin)
		var mids map[string]string
		if err := e.Info.AllMids(ctx, dex, &mids); err != nil {
			return 0, err
		}
		parsed, err := strconv.ParseFloat(mids[coin], 64)
		if err != nil {
			return 0, err
		}
		price = parsed
	} else {
		price = *px
	}
	asset := e.Info.CoinToAsset[coin]
	isSpot := asset >= 10000
	if isBuy {
		price *= 1 + slippage
	} else {
		price *= 1 - slippage
	}
	sig, err := strconv.ParseFloat(strconv.FormatFloat(price, 'g', 5, 64), 64)
	if err != nil {
		return 0, err
	}
	places := 6 - e.Info.AssetToSzDecimals[asset]
	if isSpot {
		places = 8 - e.Info.AssetToSzDecimals[asset]
	}
	return roundPlaces(sig, places), nil
}

func roundPlaces(x float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(x*factor) / factor
}

// SetExpiresAfter corresponds to Python:
// hyperliquid.exchange.Exchange.set_expires_after
func (e *Exchange) SetExpiresAfter(expiresAfter *int64) {
	e.ExpiresAfter = expiresAfter
}

// PostAction corresponds to Python:
// hyperliquid.exchange.Exchange._post_action
func (e *Exchange) PostAction(ctx context.Context, action any, signature signing.Signature, nonce int64, out any) error {
	vaultAddress := e.VaultAddress
	if actionType(action) == "usdClassTransfer" || actionType(action) == "sendAsset" {
		vaultAddress = nil
	}
	payload := map[string]any{
		"action":       action,
		"nonce":        nonce,
		"signature":    signature,
		"vaultAddress": vaultAddress,
		"expiresAfter": e.ExpiresAfter,
	}
	return e.Post(ctx, "/exchange", payload, out)
}

func actionType(action any) string {
	switch v := action.(type) {
	case signing.OrderedMap:
		if value, ok := v.Get("type"); ok {
			if s, ok := value.(string); ok {
				return s
			}
		}
	case map[string]any:
		if s, ok := v["type"].(string); ok {
			return s
		}
	}
	return ""
}

// OrderAction corresponds to the action-building part of Python:
// hyperliquid.exchange.Exchange.bulk_orders
func OrderAction(orderRequests []signing.OrderRequest, nameToAsset func(string) int, builder *types.BuilderInfo, grouping any) (signing.OrderedMap, error) {
	orderWires := make([]signing.OrderWire, 0, len(orderRequests))
	for _, order := range orderRequests {
		wire, err := signing.OrderRequestToOrderWire(order, nameToAsset(order.Coin))
		if err != nil {
			return nil, err
		}
		orderWires = append(orderWires, wire)
	}
	if builder != nil {
		builder.B = strings.ToLower(builder.B)
	}
	return signing.OrderWiresToOrderAction(orderWires, builder, grouping), nil
}

// Order corresponds to Python:
// hyperliquid.exchange.Exchange.order
func (e *Exchange) Order(ctx context.Context, name string, isBuy bool, sz float64, limitPx float64, orderType signing.OrderType, reduceOnly bool, cloid *types.Cloid, builder *types.BuilderInfo, out any) error {
	order := signing.OrderRequest{
		Coin:       name,
		IsBuy:      isBuy,
		Sz:         sz,
		LimitPx:    limitPx,
		OrderType:  orderType,
		ReduceOnly: reduceOnly,
		Cloid:      cloid,
	}
	return e.BulkOrders(ctx, []signing.OrderRequest{order}, builder, "na", out)
}

// BulkOrders corresponds to Python:
// hyperliquid.exchange.Exchange.bulk_orders
func (e *Exchange) BulkOrders(ctx context.Context, orderRequests []signing.OrderRequest, builder *types.BuilderInfo, grouping any, out any) error {
	timestamp := signing.GetTimestampMs()
	action, err := OrderAction(orderRequests, e.Info.NameToAsset, builder, grouping)
	if err != nil {
		return err
	}
	signature, err := signing.SignL1Action(e.Wallet, action, e.VaultAddress, timestamp, e.ExpiresAfter, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, action, signature, timestamp, out)
}

// MarketOpen corresponds to Python:
// hyperliquid.exchange.Exchange.market_open
func (e *Exchange) MarketOpen(ctx context.Context, name string, isBuy bool, sz float64, px *float64, slippage float64, cloid *types.Cloid, builder *types.BuilderInfo, out any) error {
	limitPx, err := e.SlippagePrice(ctx, name, isBuy, slippage, px)
	if err != nil {
		return err
	}
	return e.Order(ctx, name, isBuy, sz, limitPx, signing.OrderType{"limit": map[string]any{"tif": "Ioc"}}, false, cloid, builder, out)
}

// MarketClose corresponds to Python:
// hyperliquid.exchange.Exchange.market_close
func (e *Exchange) MarketClose(ctx context.Context, coin string, sz *float64, px *float64, slippage float64, cloid *types.Cloid, builder *types.BuilderInfo, out any) error {
	address := crypto.PubkeyToAddress(e.Wallet.PublicKey).Hex()
	if e.AccountAddress != nil {
		address = *e.AccountAddress
	}
	if e.VaultAddress != nil {
		address = *e.VaultAddress
	}
	dex := GetDex(coin)
	var state map[string]any
	if err := e.Info.UserState(ctx, address, dex, &state); err != nil {
		return err
	}
	positions, _ := state["assetPositions"].([]any)
	for _, rawPosition := range positions {
		position, _ := rawPosition.(map[string]any)
		item, _ := position["position"].(map[string]any)
		if item["coin"] != coin {
			continue
		}
		szi, err := strconv.ParseFloat(fmtAnyString(item["szi"]), 64)
		if err != nil {
			return err
		}
		closeSz := math.Abs(szi)
		if sz != nil {
			closeSz = *sz
		}
		isBuy := szi < 0
		limitPx, err := e.SlippagePrice(ctx, coin, isBuy, slippage, px)
		if err != nil {
			return err
		}
		return e.Order(ctx, coin, isBuy, closeSz, limitPx, signing.OrderType{"limit": map[string]any{"tif": "Ioc"}}, true, cloid, builder, out)
	}
	return nil
}

func fmtAnyString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

// Cancel corresponds to Python:
// hyperliquid.exchange.Exchange.cancel
func (e *Exchange) Cancel(ctx context.Context, name string, oid int, out any) error {
	return e.BulkCancel(ctx, []CancelRequest{{Coin: name, OID: oid}}, out)
}

// CancelRequest mirrors hyperliquid.utils.signing.CancelRequest.
type CancelRequest struct {
	Coin string
	OID  int
}

// BulkCancel corresponds to Python:
// hyperliquid.exchange.Exchange.bulk_cancel
func (e *Exchange) BulkCancel(ctx context.Context, cancelRequests []CancelRequest, out any) error {
	timestamp := signing.GetTimestampMs()
	cancels := make([]signing.OrderedMap, 0, len(cancelRequests))
	for _, cancel := range cancelRequests {
		cancels = append(cancels, signing.OrderedMap{
			{Key: "a", Value: e.Info.NameToAsset(cancel.Coin)},
			{Key: "o", Value: cancel.OID},
		})
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "cancel"},
		{Key: "cancels", Value: cancels},
	}
	signature, err := signing.SignL1Action(e.Wallet, action, e.VaultAddress, timestamp, e.ExpiresAfter, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, action, signature, timestamp, out)
}

// ScheduleCancel corresponds to Python:
// hyperliquid.exchange.Exchange.schedule_cancel
func (e *Exchange) ScheduleCancel(ctx context.Context, cancelTime *int64, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "scheduleCancel"}}
	if cancelTime != nil {
		action = append(action, signing.Field{Key: "time", Value: *cancelTime})
	}
	signature, err := signing.SignL1Action(e.Wallet, action, e.VaultAddress, timestamp, e.ExpiresAfter, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, action, signature, timestamp, out)
}

func (e *Exchange) signAndPostL1(ctx context.Context, action signing.OrderedMap, vaultAddress *string, nonce int64, out any) error {
	signature, err := signing.SignL1Action(e.Wallet, action, vaultAddress, nonce, e.ExpiresAfter, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, action, signature, nonce, out)
}

func floatString(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// OidOrCloid mirrors hyperliquid.utils.signing.OidOrCloid.
type OidOrCloid any

// ModifyRequest mirrors hyperliquid.utils.signing.ModifyRequest.
type ModifyRequest struct {
	OID   OidOrCloid
	Order signing.OrderRequest
}

// ModifyOrder corresponds to Python:
// hyperliquid.exchange.Exchange.modify_order
func (e *Exchange) ModifyOrder(ctx context.Context, oid OidOrCloid, name string, isBuy bool, sz float64, limitPx float64, orderType signing.OrderType, reduceOnly bool, cloid *types.Cloid, out any) error {
	return e.BulkModifyOrdersNew(ctx, []ModifyRequest{{
		OID: oid,
		Order: signing.OrderRequest{
			Coin:       name,
			IsBuy:      isBuy,
			Sz:         sz,
			LimitPx:    limitPx,
			OrderType:  orderType,
			ReduceOnly: reduceOnly,
			Cloid:      cloid,
		},
	}}, out)
}

// BulkModifyOrdersNew corresponds to Python:
// hyperliquid.exchange.Exchange.bulk_modify_orders_new
func (e *Exchange) BulkModifyOrdersNew(ctx context.Context, modifyRequests []ModifyRequest, out any) error {
	timestamp := signing.GetTimestampMs()
	modifies := make([]signing.OrderedMap, 0, len(modifyRequests))
	for _, modify := range modifyRequests {
		wire, err := signing.OrderRequestToOrderWire(modify.Order, e.Info.NameToAsset(modify.Order.Coin))
		if err != nil {
			return err
		}
		modifies = append(modifies, signing.OrderedMap{
			{Key: "oid", Value: oidToRaw(modify.OID)},
			{Key: "order", Value: wire},
		})
	}
	action := signing.OrderedMap{{Key: "type", Value: "batchModify"}, {Key: "modifies", Value: modifies}}
	return e.signAndPostL1(ctx, action, e.VaultAddress, timestamp, out)
}

func oidToRaw(oid OidOrCloid) any {
	switch v := oid.(type) {
	case types.Cloid:
		return v.ToRaw()
	case *types.Cloid:
		return v.ToRaw()
	default:
		return v
	}
}

// CancelByCloidRequest mirrors hyperliquid.utils.signing.CancelByCloidRequest.
type CancelByCloidRequest struct {
	Coin  string
	Cloid types.Cloid
}

// CancelByCloid corresponds to Python:
// hyperliquid.exchange.Exchange.cancel_by_cloid
func (e *Exchange) CancelByCloid(ctx context.Context, name string, cloid types.Cloid, out any) error {
	return e.BulkCancelByCloid(ctx, []CancelByCloidRequest{{Coin: name, Cloid: cloid}}, out)
}

// BulkCancelByCloid corresponds to Python:
// hyperliquid.exchange.Exchange.bulk_cancel_by_cloid
func (e *Exchange) BulkCancelByCloid(ctx context.Context, cancelRequests []CancelByCloidRequest, out any) error {
	timestamp := signing.GetTimestampMs()
	cancels := make([]signing.OrderedMap, 0, len(cancelRequests))
	for _, cancel := range cancelRequests {
		cancels = append(cancels, signing.OrderedMap{
			{Key: "asset", Value: e.Info.NameToAsset(cancel.Coin)},
			{Key: "cloid", Value: cancel.Cloid.ToRaw()},
		})
	}
	action := signing.OrderedMap{{Key: "type", Value: "cancelByCloid"}, {Key: "cancels", Value: cancels}}
	return e.signAndPostL1(ctx, action, e.VaultAddress, timestamp, out)
}

// UpdateLeverage corresponds to Python:
// hyperliquid.exchange.Exchange.update_leverage
func (e *Exchange) UpdateLeverage(ctx context.Context, leverage int, name string, isCross bool, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "updateLeverage"},
		{Key: "asset", Value: e.Info.NameToAsset(name)},
		{Key: "isCross", Value: isCross},
		{Key: "leverage", Value: leverage},
	}
	return e.signAndPostL1(ctx, action, e.VaultAddress, timestamp, out)
}

// UpdateIsolatedMargin corresponds to Python:
// hyperliquid.exchange.Exchange.update_isolated_margin
func (e *Exchange) UpdateIsolatedMargin(ctx context.Context, amount float64, name string, out any) error {
	timestamp := signing.GetTimestampMs()
	ntli, err := signing.FloatToUSDInt(amount)
	if err != nil {
		return err
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "updateIsolatedMargin"},
		{Key: "asset", Value: e.Info.NameToAsset(name)},
		{Key: "isBuy", Value: true},
		{Key: "ntli", Value: ntli},
	}
	return e.signAndPostL1(ctx, action, e.VaultAddress, timestamp, out)
}

// SetReferrer corresponds to Python:
// hyperliquid.exchange.Exchange.set_referrer
func (e *Exchange) SetReferrer(ctx context.Context, code string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "setReferrer"}, {Key: "code", Value: code}}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// CreateSubAccount corresponds to Python:
// hyperliquid.exchange.Exchange.create_sub_account
func (e *Exchange) CreateSubAccount(ctx context.Context, name string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "createSubAccount"}, {Key: "name", Value: name}}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// USDClassTransfer corresponds to Python:
// hyperliquid.exchange.Exchange.usd_class_transfer
func (e *Exchange) USDClassTransfer(ctx context.Context, amount float64, toPerp bool, out any) error {
	timestamp := signing.GetTimestampMs()
	strAmount := floatString(amount)
	if e.VaultAddress != nil {
		strAmount += " subaccount:" + *e.VaultAddress
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "usdClassTransfer"},
		{Key: "amount", Value: strAmount},
		{Key: "toPerp", Value: toPerp},
		{Key: "nonce", Value: uint64(timestamp)},
	}
	signature, signedAction, err := signing.SignUSDClassTransferAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// SendAsset corresponds to Python:
// hyperliquid.exchange.Exchange.send_asset
func (e *Exchange) SendAsset(ctx context.Context, destination string, sourceDex string, destinationDex string, token string, amount float64, out any) error {
	timestamp := signing.GetTimestampMs()
	fromSubAccount := ""
	if e.VaultAddress != nil {
		fromSubAccount = *e.VaultAddress
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "sendAsset"},
		{Key: "destination", Value: destination},
		{Key: "sourceDex", Value: sourceDex},
		{Key: "destinationDex", Value: destinationDex},
		{Key: "token", Value: token},
		{Key: "amount", Value: floatString(amount)},
		{Key: "fromSubAccount", Value: fromSubAccount},
		{Key: "nonce", Value: uint64(timestamp)},
	}
	signature, signedAction, err := signing.SignSendAssetAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// SubAccountTransfer corresponds to Python:
// hyperliquid.exchange.Exchange.sub_account_transfer
func (e *Exchange) SubAccountTransfer(ctx context.Context, subAccountUser string, isDeposit bool, usd int, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "subAccountTransfer"},
		{Key: "subAccountUser", Value: subAccountUser},
		{Key: "isDeposit", Value: isDeposit},
		{Key: "usd", Value: usd},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SubAccountSpotTransfer corresponds to Python:
// hyperliquid.exchange.Exchange.sub_account_spot_transfer
func (e *Exchange) SubAccountSpotTransfer(ctx context.Context, subAccountUser string, isDeposit bool, token string, amount float64, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "subAccountSpotTransfer"},
		{Key: "subAccountUser", Value: subAccountUser},
		{Key: "isDeposit", Value: isDeposit},
		{Key: "token", Value: token},
		{Key: "amount", Value: floatString(amount)},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// VaultUSDTransfer corresponds to Python:
// hyperliquid.exchange.Exchange.vault_usd_transfer
func (e *Exchange) VaultUSDTransfer(ctx context.Context, vaultAddress string, isDeposit bool, usd int, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "vaultTransfer"},
		{Key: "vaultAddress", Value: vaultAddress},
		{Key: "isDeposit", Value: isDeposit},
		{Key: "usd", Value: usd},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// USDTransfer corresponds to Python:
// hyperliquid.exchange.Exchange.usd_transfer
func (e *Exchange) USDTransfer(ctx context.Context, amount float64, destination string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "destination", Value: destination},
		{Key: "amount", Value: floatString(amount)},
		{Key: "time", Value: uint64(timestamp)},
		{Key: "type", Value: "usdSend"},
	}
	signature, signedAction, err := signing.SignUSDTransferAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// SpotTransfer corresponds to Python:
// hyperliquid.exchange.Exchange.spot_transfer
func (e *Exchange) SpotTransfer(ctx context.Context, amount float64, destination string, token string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "destination", Value: destination},
		{Key: "amount", Value: floatString(amount)},
		{Key: "token", Value: token},
		{Key: "time", Value: uint64(timestamp)},
		{Key: "type", Value: "spotSend"},
	}
	signature, signedAction, err := signing.SignSpotTransferAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// TokenDelegate corresponds to Python:
// hyperliquid.exchange.Exchange.token_delegate
func (e *Exchange) TokenDelegate(ctx context.Context, validator string, wei uint64, isUndelegate bool, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "validator", Value: validator},
		{Key: "wei", Value: wei},
		{Key: "isUndelegate", Value: isUndelegate},
		{Key: "nonce", Value: uint64(timestamp)},
		{Key: "type", Value: "tokenDelegate"},
	}
	signature, signedAction, err := signing.SignTokenDelegateAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// WithdrawFromBridge corresponds to Python:
// hyperliquid.exchange.Exchange.withdraw_from_bridge
func (e *Exchange) WithdrawFromBridge(ctx context.Context, amount float64, destination string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "destination", Value: destination},
		{Key: "amount", Value: floatString(amount)},
		{Key: "time", Value: uint64(timestamp)},
		{Key: "type", Value: "withdraw3"},
	}
	signature, signedAction, err := signing.SignWithdrawFromBridgeAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// ApproveAgent corresponds to Python:
// hyperliquid.exchange.Exchange.approve_agent
func (e *Exchange) ApproveAgent(ctx context.Context, name *string, out any) (string, error) {
	agentKey, err := crypto.GenerateKey()
	if err != nil {
		return "", err
	}
	agentKeyHex := "0x" + hex.EncodeToString(crypto.FromECDSA(agentKey))
	agentAddress := crypto.PubkeyToAddress(agentKey.PublicKey).Hex()
	timestamp := signing.GetTimestampMs()
	agentName := ""
	if name != nil {
		agentName = *name
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "approveAgent"},
		{Key: "agentAddress", Value: agentAddress},
		{Key: "agentName", Value: agentName},
		{Key: "nonce", Value: uint64(timestamp)},
	}
	signature, signedAction, err := signing.SignAgent(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return "", err
	}
	if name == nil {
		signedAction = removeField(signedAction, "agentName")
	}
	return agentKeyHex, e.PostAction(ctx, signedAction, signature, timestamp, out)
}

func removeField(action signing.OrderedMap, key string) signing.OrderedMap {
	out := make(signing.OrderedMap, 0, len(action))
	for _, field := range action {
		if field.Key != key {
			out = append(out, field)
		}
	}
	return out
}

// ApproveBuilderFee corresponds to Python:
// hyperliquid.exchange.Exchange.approve_builder_fee
func (e *Exchange) ApproveBuilderFee(ctx context.Context, builder string, maxFeeRate string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "maxFeeRate", Value: maxFeeRate},
		{Key: "builder", Value: builder},
		{Key: "nonce", Value: uint64(timestamp)},
		{Key: "type", Value: "approveBuilderFee"},
	}
	signature, signedAction, err := signing.SignApproveBuilderFee(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// ConvertToMultiSigUser corresponds to Python:
// hyperliquid.exchange.Exchange.convert_to_multi_sig_user
func (e *Exchange) ConvertToMultiSigUser(ctx context.Context, authorizedUsers []string, threshold int, out any) error {
	timestamp := signing.GetTimestampMs()
	sort.Strings(authorizedUsers)
	signers := map[string]any{"authorizedUsers": authorizedUsers, "threshold": threshold}
	signersJSON, err := json.Marshal(signers)
	if err != nil {
		return err
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "convertToMultiSigUser"},
		{Key: "signers", Value: string(signersJSON)},
		{Key: "nonce", Value: uint64(timestamp)},
	}
	signature, signedAction, err := signing.SignConvertToMultiSigUserAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// MultiSig corresponds to Python:
// hyperliquid.exchange.Exchange.multi_sig
func (e *Exchange) MultiSig(ctx context.Context, multiSigUser string, innerAction any, signatures any, nonce int64, vaultAddress *string, out any) error {
	multiSigUser = strings.ToLower(multiSigUser)
	outerSigner := strings.ToLower(crypto.PubkeyToAddress(e.Wallet.PublicKey).Hex())
	multiSigAction := signing.OrderedMap{
		{Key: "type", Value: "multiSig"},
		{Key: "signatureChainId", Value: "0x66eee"},
		{Key: "signatures", Value: signatures},
		{Key: "payload", Value: signing.OrderedMap{
			{Key: "multiSigUser", Value: multiSigUser},
			{Key: "outerSigner", Value: outerSigner},
			{Key: "action", Value: innerAction},
		}},
	}
	signature, err := signing.SignMultiSigAction(e.Wallet, multiSigAction, e.IsMainnet, vaultAddress, nonce, e.ExpiresAfter)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, multiSigAction, signature, nonce, out)
}

// UseBigBlocks corresponds to Python:
// hyperliquid.exchange.Exchange.use_big_blocks
func (e *Exchange) UseBigBlocks(ctx context.Context, enable bool, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "evmUserModify"}, {Key: "usingBigBlocks", Value: enable}}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// AgentEnableDexAbstraction corresponds to Python:
// hyperliquid.exchange.Exchange.agent_enable_dex_abstraction
func (e *Exchange) AgentEnableDexAbstraction(ctx context.Context, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "agentEnableDexAbstraction"}}
	return e.signAndPostL1(ctx, action, e.VaultAddress, timestamp, out)
}

// AgentSetAbstraction corresponds to Python:
// hyperliquid.exchange.Exchange.agent_set_abstraction
func (e *Exchange) AgentSetAbstraction(ctx context.Context, abstraction string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "agentSetAbstraction"}, {Key: "abstraction", Value: abstraction}}
	return e.signAndPostL1(ctx, action, e.VaultAddress, timestamp, out)
}

// UserDexAbstraction corresponds to Python:
// hyperliquid.exchange.Exchange.user_dex_abstraction
func (e *Exchange) UserDexAbstraction(ctx context.Context, user string, enabled bool, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "userDexAbstraction"},
		{Key: "user", Value: strings.ToLower(user)},
		{Key: "enabled", Value: enabled},
		{Key: "nonce", Value: uint64(timestamp)},
	}
	signature, signedAction, err := signing.SignUserDexAbstractionAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// UserSetAbstraction corresponds to Python:
// hyperliquid.exchange.Exchange.user_set_abstraction
func (e *Exchange) UserSetAbstraction(ctx context.Context, user string, abstraction string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "userSetAbstraction"},
		{Key: "user", Value: strings.ToLower(user)},
		{Key: "abstraction", Value: abstraction},
		{Key: "nonce", Value: uint64(timestamp)},
	}
	signature, signedAction, err := signing.SignUserSetAbstractionAction(e.Wallet, action, e.IsMainnet)
	if err != nil {
		return err
	}
	return e.PostAction(ctx, signedAction, signature, timestamp, out)
}

// Noop corresponds to Python:
// hyperliquid.exchange.Exchange.noop
func (e *Exchange) Noop(ctx context.Context, nonce int64, out any) error {
	action := signing.OrderedMap{{Key: "type", Value: "noop"}}
	return e.signAndPostL1(ctx, action, e.VaultAddress, nonce, out)
}

// GossipPriorityBid corresponds to Python:
// hyperliquid.exchange.Exchange.gossip_priority_bid
func (e *Exchange) GossipPriorityBid(ctx context.Context, slotID int, ip string, maxGas int, out any) error {
	nonce := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "gossipPriorityBid"},
		{Key: "slotId", Value: slotID},
		{Key: "ip", Value: ip},
		{Key: "maxGas", Value: maxGas},
	}
	return e.signAndPostL1(ctx, action, e.VaultAddress, nonce, out)
}

// SpotDeployRegisterToken corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_register_token
func (e *Exchange) SpotDeployRegisterToken(ctx context.Context, tokenName string, szDecimals int, weiDecimals int, maxGas int, fullName string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "spotDeploy"},
		{Key: "registerToken2", Value: signing.OrderedMap{
			{Key: "spec", Value: signing.OrderedMap{
				{Key: "name", Value: tokenName},
				{Key: "szDecimals", Value: szDecimals},
				{Key: "weiDecimals", Value: weiDecimals},
			}},
			{Key: "maxGas", Value: maxGas},
			{Key: "fullName", Value: fullName},
		}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SpotDeployUserGenesis corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_user_genesis
func (e *Exchange) SpotDeployUserGenesis(ctx context.Context, token int, userAndWei [][]string, existingTokenAndWei []any, out any) error {
	timestamp := signing.GetTimestampMs()
	lowered := make([][]string, 0, len(userAndWei))
	for _, pair := range userAndWei {
		if len(pair) >= 2 {
			lowered = append(lowered, []string{strings.ToLower(pair[0]), pair[1]})
		}
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "spotDeploy"},
		{Key: "userGenesis", Value: signing.OrderedMap{
			{Key: "token", Value: token},
			{Key: "userAndWei", Value: lowered},
			{Key: "existingTokenAndWei", Value: existingTokenAndWei},
		}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SpotDeployEnableFreezePrivilege corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_enable_freeze_privilege
func (e *Exchange) SpotDeployEnableFreezePrivilege(ctx context.Context, token int, out any) error {
	return e.SpotDeployTokenActionInner(ctx, "enableFreezePrivilege", token, out)
}

// SpotDeployFreezeUser corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_freeze_user
func (e *Exchange) SpotDeployFreezeUser(ctx context.Context, token int, user string, freeze bool, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "spotDeploy"},
		{Key: "freezeUser", Value: signing.OrderedMap{
			{Key: "token", Value: token},
			{Key: "user", Value: strings.ToLower(user)},
			{Key: "freeze", Value: freeze},
		}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SpotDeployRevokeFreezePrivilege corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_revoke_freeze_privilege
func (e *Exchange) SpotDeployRevokeFreezePrivilege(ctx context.Context, token int, out any) error {
	return e.SpotDeployTokenActionInner(ctx, "revokeFreezePrivilege", token, out)
}

// SpotDeployEnableQuoteToken corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_enable_quote_token
func (e *Exchange) SpotDeployEnableQuoteToken(ctx context.Context, token int, out any) error {
	return e.SpotDeployTokenActionInner(ctx, "enableQuoteToken", token, out)
}

// SpotDeployTokenActionInner corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_token_action_inner
func (e *Exchange) SpotDeployTokenActionInner(ctx context.Context, variant string, token int, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "spotDeploy"},
		{Key: variant, Value: signing.OrderedMap{{Key: "token", Value: token}}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SpotDeployGenesis corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_genesis
func (e *Exchange) SpotDeployGenesis(ctx context.Context, token int, maxSupply string, noHyperliquidity bool, out any) error {
	timestamp := signing.GetTimestampMs()
	genesis := signing.OrderedMap{{Key: "token", Value: token}, {Key: "maxSupply", Value: maxSupply}}
	if noHyperliquidity {
		genesis = append(genesis, signing.Field{Key: "noHyperliquidity", Value: true})
	}
	action := signing.OrderedMap{{Key: "type", Value: "spotDeploy"}, {Key: "genesis", Value: genesis}}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SpotDeployRegisterSpot corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_register_spot
func (e *Exchange) SpotDeployRegisterSpot(ctx context.Context, baseToken int, quoteToken int, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "spotDeploy"},
		{Key: "registerSpot", Value: signing.OrderedMap{{Key: "tokens", Value: []int{baseToken, quoteToken}}}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SpotDeployRegisterHyperliquidity corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_register_hyperliquidity
func (e *Exchange) SpotDeployRegisterHyperliquidity(ctx context.Context, spot int, startPx float64, orderSz float64, nOrders int, nSeededLevels *int, out any) error {
	timestamp := signing.GetTimestampMs()
	register := signing.OrderedMap{
		{Key: "spot", Value: spot},
		{Key: "startPx", Value: floatString(startPx)},
		{Key: "orderSz", Value: floatString(orderSz)},
		{Key: "nOrders", Value: nOrders},
	}
	if nSeededLevels != nil {
		register = append(register, signing.Field{Key: "nSeededLevels", Value: *nSeededLevels})
	}
	action := signing.OrderedMap{{Key: "type", Value: "spotDeploy"}, {Key: "registerHyperliquidity", Value: register}}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// SpotDeploySetDeployerTradingFeeShare corresponds to Python:
// hyperliquid.exchange.Exchange.spot_deploy_set_deployer_trading_fee_share
func (e *Exchange) SpotDeploySetDeployerTradingFeeShare(ctx context.Context, token int, share string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "spotDeploy"},
		{Key: "setDeployerTradingFeeShare", Value: signing.OrderedMap{{Key: "token", Value: token}, {Key: "share", Value: share}}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// PerpDeployRegisterAsset corresponds to Python:
// hyperliquid.exchange.Exchange.perp_deploy_register_asset
func (e *Exchange) PerpDeployRegisterAsset(ctx context.Context, dex string, maxGas *int, coin string, szDecimals int, oraclePx string, marginTableID int, onlyIsolated bool, schema *types.PerpDexSchemaInput, out any) error {
	timestamp := signing.GetTimestampMs()
	var schemaWire any
	if schema != nil {
		var oracleUpdater any
		if schema.OracleUpdater != nil {
			oracleUpdater = strings.ToLower(*schema.OracleUpdater)
		}
		schemaWire = signing.OrderedMap{
			{Key: "fullName", Value: schema.FullName},
			{Key: "collateralToken", Value: schema.CollateralToken},
			{Key: "oracleUpdater", Value: oracleUpdater},
		}
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "perpDeploy"},
		{Key: "registerAsset", Value: signing.OrderedMap{
			{Key: "maxGas", Value: maxGas},
			{Key: "assetRequest", Value: signing.OrderedMap{
				{Key: "coin", Value: coin},
				{Key: "szDecimals", Value: szDecimals},
				{Key: "oraclePx", Value: oraclePx},
				{Key: "marginTableId", Value: marginTableID},
				{Key: "onlyIsolated", Value: onlyIsolated},
			}},
			{Key: "dex", Value: dex},
			{Key: "schema", Value: schemaWire},
		}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// PerpDeploySetOracle corresponds to Python:
// hyperliquid.exchange.Exchange.perp_deploy_set_oracle
func (e *Exchange) PerpDeploySetOracle(ctx context.Context, dex string, oraclePxs map[string]string, allMarkPxs []map[string]string, externalPerpPxs map[string]string, out any) error {
	timestamp := signing.GetTimestampMs()
	markPxsWire := make([][][]string, 0, len(allMarkPxs))
	for _, markPxs := range allMarkPxs {
		markPxsWire = append(markPxsWire, sortedStringPairs(markPxs))
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "perpDeploy"},
		{Key: "setOracle", Value: signing.OrderedMap{
			{Key: "dex", Value: dex},
			{Key: "oraclePxs", Value: sortedStringPairs(oraclePxs)},
			{Key: "markPxs", Value: markPxsWire},
			{Key: "externalPerpPxs", Value: sortedStringPairs(externalPerpPxs)},
		}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

func sortedStringPairs(values map[string]string) [][]string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([][]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, []string{key, values[key]})
	}
	return pairs
}

// CSignerUnjailSelf corresponds to Python:
// hyperliquid.exchange.Exchange.c_signer_unjail_self
func (e *Exchange) CSignerUnjailSelf(ctx context.Context, out any) error {
	return e.CSignerInner(ctx, "unjailSelf", out)
}

// CSignerJailSelf corresponds to Python:
// hyperliquid.exchange.Exchange.c_signer_jail_self
func (e *Exchange) CSignerJailSelf(ctx context.Context, out any) error {
	return e.CSignerInner(ctx, "jailSelf", out)
}

// CSignerInner corresponds to Python:
// hyperliquid.exchange.Exchange.c_signer_inner
func (e *Exchange) CSignerInner(ctx context.Context, variant string, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "CSignerAction"}, {Key: variant, Value: nil}}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// CValidatorRegister corresponds to Python:
// hyperliquid.exchange.Exchange.c_validator_register
func (e *Exchange) CValidatorRegister(ctx context.Context, nodeIP string, name string, description string, delegationsDisabled bool, commissionBps int, signer string, unjailed bool, initialWei uint64, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{
		{Key: "type", Value: "CValidatorAction"},
		{Key: "register", Value: signing.OrderedMap{
			{Key: "profile", Value: signing.OrderedMap{
				{Key: "node_ip", Value: signing.OrderedMap{{Key: "Ip", Value: nodeIP}}},
				{Key: "name", Value: name},
				{Key: "description", Value: description},
				{Key: "delegations_disabled", Value: delegationsDisabled},
				{Key: "commission_bps", Value: commissionBps},
				{Key: "signer", Value: signer},
			}},
			{Key: "unjailed", Value: unjailed},
			{Key: "initial_wei", Value: initialWei},
		}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

// CValidatorChangeProfile corresponds to Python:
// hyperliquid.exchange.Exchange.c_validator_change_profile
func (e *Exchange) CValidatorChangeProfile(ctx context.Context, nodeIP *string, name *string, description *string, unjailed bool, disableDelegations *bool, commissionBps *int, signer *string, out any) error {
	timestamp := signing.GetTimestampMs()
	var nodeIPWire any
	if nodeIP != nil {
		nodeIPWire = signing.OrderedMap{{Key: "Ip", Value: *nodeIP}}
	}
	action := signing.OrderedMap{
		{Key: "type", Value: "CValidatorAction"},
		{Key: "changeProfile", Value: signing.OrderedMap{
			{Key: "node_ip", Value: nodeIPWire},
			{Key: "name", Value: stringPtrValue(name)},
			{Key: "description", Value: stringPtrValue(description)},
			{Key: "unjailed", Value: unjailed},
			{Key: "disable_delegations", Value: boolPtrValue(disableDelegations)},
			{Key: "commission_bps", Value: intPtrValue(commissionBps)},
			{Key: "signer", Value: stringPtrValue(signer)},
		}},
	}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}

func stringPtrValue(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func boolPtrValue(v *bool) any {
	if v == nil {
		return nil
	}
	return *v
}

func intPtrValue(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

// CValidatorUnregister corresponds to Python:
// hyperliquid.exchange.Exchange.c_validator_unregister
func (e *Exchange) CValidatorUnregister(ctx context.Context, out any) error {
	timestamp := signing.GetTimestampMs()
	action := signing.OrderedMap{{Key: "type", Value: "CValidatorAction"}, {Key: "unregister", Value: nil}}
	return e.signAndPostL1(ctx, action, nil, timestamp, out)
}
