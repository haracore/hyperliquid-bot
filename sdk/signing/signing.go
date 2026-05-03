// Package signing mirrors hyperliquid-python-sdk/hyperliquid/utils/signing.py.
package signing

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"hyperliquid-bot/sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"
)

// OrderType mirrors hyperliquid.utils.signing.OrderType.
type OrderType map[string]any

// OrderRequest mirrors hyperliquid.utils.signing.OrderRequest.
type OrderRequest struct {
	Coin       string
	IsBuy      bool
	Sz         float64
	LimitPx    float64
	OrderType  OrderType
	ReduceOnly bool
	Cloid      *types.Cloid
}

// Field is one ordered key/value entry in a Python dict-compatible payload.
type Field struct {
	Key   string
	Value any
}

// OrderedMap mirrors Python dict insertion order for msgpack signing payloads.
type OrderedMap []Field

// OrderWire mirrors hyperliquid.utils.signing.OrderWire.
type OrderWire OrderedMap

// Signature mirrors the Python dict returned by signing helpers.
type Signature struct {
	R string `json:"r"`
	S string `json:"s"`
	V int    `json:"v"`
}

// SignType mirrors the Python sign type dictionaries.
type SignType struct {
	Name string
	Type string
}

var (
	// USDSendSignTypes corresponds to Python USD_SEND_SIGN_TYPES.
	USDSendSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	// WithdrawSignTypes corresponds to Python WITHDRAW_SIGN_TYPES.
	WithdrawSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	// SpotTransferSignTypes corresponds to Python SPOT_TRANSFER_SIGN_TYPES.
	SpotTransferSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "time", Type: "uint64"},
	}

	// USDClassTransferSignTypes corresponds to Python USD_CLASS_TRANSFER_SIGN_TYPES.
	USDClassTransferSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "toPerp", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	// SendAssetSignTypes corresponds to Python SEND_ASSET_SIGN_TYPES.
	SendAssetSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "destination", Type: "string"},
		{Name: "sourceDex", Type: "string"},
		{Name: "destinationDex", Type: "string"},
		{Name: "token", Type: "string"},
		{Name: "amount", Type: "string"},
		{Name: "fromSubAccount", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	// UserDexAbstractionSignTypes corresponds to Python USER_DEX_ABSTRACTION_SIGN_TYPES.
	UserDexAbstractionSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "user", Type: "address"},
		{Name: "enabled", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	// UserSetAbstractionSignTypes corresponds to Python USER_SET_ABSTRACTION_SIGN_TYPES.
	UserSetAbstractionSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "user", Type: "address"},
		{Name: "abstraction", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	// TokenDelegateSignTypes corresponds to Python TOKEN_DELEGATE_TYPES.
	TokenDelegateSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "validator", Type: "address"},
		{Name: "wei", Type: "uint64"},
		{Name: "isUndelegate", Type: "bool"},
		{Name: "nonce", Type: "uint64"},
	}

	// ConvertToMultiSigUserSignTypes corresponds to Python CONVERT_TO_MULTI_SIG_USER_SIGN_TYPES.
	ConvertToMultiSigUserSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "signers", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}

	// MultiSigEnvelopeSignTypes corresponds to Python MULTI_SIG_ENVELOPE_SIGN_TYPES.
	MultiSigEnvelopeSignTypes = []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "multiSigActionHash", Type: "bytes32"},
		{Name: "nonce", Type: "uint64"},
	}
)

// PrivateKeyFromHex is a Go helper for examples and tests. Python callers use:
// eth_account.Account.from_key
func PrivateKeyFromHex(privateKey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(strings.TrimPrefix(privateKey, "0x"))
}

// FloatToWire corresponds to Python:
// hyperliquid.utils.signing.float_to_wire
func FloatToWire(x float64) (string, error) {
	rounded := fmt.Sprintf("%.8f", x)
	parsed, err := strconv.ParseFloat(rounded, 64)
	if err != nil {
		return "", err
	}
	if math.Abs(parsed-x) >= 1e-12 {
		return "", fmt.Errorf("float_to_wire causes rounding: %v", x)
	}
	if rounded == "-0" {
		rounded = "0"
	}
	rounded = strings.TrimRight(rounded, "0")
	rounded = strings.TrimRight(rounded, ".")
	if rounded == "" || rounded == "-0" {
		return "0", nil
	}
	return rounded, nil
}

// FloatToIntForHashing corresponds to Python:
// hyperliquid.utils.signing.float_to_int_for_hashing
func FloatToIntForHashing(x float64) (*big.Int, error) {
	return FloatToInt(x, 8)
}

// FloatToUSDInt corresponds to Python:
// hyperliquid.utils.signing.float_to_usd_int
func FloatToUSDInt(x float64) (*big.Int, error) {
	return FloatToInt(x, 6)
}

// FloatToInt corresponds to Python:
// hyperliquid.utils.signing.float_to_int
func FloatToInt(x float64, power int) (*big.Int, error) {
	decimalText := strconv.FormatFloat(x, 'f', -1, 64)
	rat, ok := new(big.Rat).SetString(decimalText)
	if !ok {
		return nil, fmt.Errorf("could not parse float decimal: %v", x)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(power)), nil)
	rat.Mul(rat, new(big.Rat).SetInt(scale))
	roundedFloat, _ := rat.Float64()
	rounded := math.Round(roundedFloat)
	if math.Abs(rounded-roundedFloat) >= 1e-3 {
		return nil, fmt.Errorf("float_to_int causes rounding: %v", x)
	}
	num := rat.Num()
	den := rat.Denom()
	quo, rem := new(big.Int).QuoRem(num, den, new(big.Int))
	twiceRem := new(big.Int).Mul(rem, big.NewInt(2))
	if twiceRem.Cmp(den) >= 0 {
		quo.Add(quo, big.NewInt(1))
	}
	return quo, nil
}

// GetTimestampMs corresponds to Python:
// hyperliquid.utils.signing.get_timestamp_ms
func GetTimestampMs() int64 {
	return time.Now().UnixMilli()
}

// AddressToBytes corresponds to Python:
// hyperliquid.utils.signing.address_to_bytes
func AddressToBytes(address string) ([]byte, error) {
	return common.FromHex(address), nil
}

// ActionHash corresponds to Python:
// hyperliquid.utils.signing.action_hash
func ActionHash(action any, vaultAddress *string, nonce int64, expiresAfter *int64) ([]byte, error) {
	data, err := msgpack.Marshal(action)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write(data)
	if err := binary.Write(&buf, binary.BigEndian, uint64(nonce)); err != nil {
		return nil, err
	}
	if vaultAddress == nil {
		buf.WriteByte(0x00)
	} else {
		buf.WriteByte(0x01)
		addrBytes, err := AddressToBytes(*vaultAddress)
		if err != nil {
			return nil, err
		}
		buf.Write(addrBytes)
	}
	if expiresAfter != nil {
		buf.WriteByte(0x00)
		if err := binary.Write(&buf, binary.BigEndian, uint64(*expiresAfter)); err != nil {
			return nil, err
		}
	}
	hash := crypto.Keccak256(buf.Bytes())
	return hash, nil
}

// ConstructPhantomAgent corresponds to Python:
// hyperliquid.utils.signing.construct_phantom_agent
func ConstructPhantomAgent(hash []byte, isMainnet bool) map[string]any {
	source := "b"
	if isMainnet {
		source = "a"
	}
	return map[string]any{
		"source":       source,
		"connectionId": hash,
	}
}

// L1Payload corresponds to Python:
// hyperliquid.utils.signing.l1_payload
func L1Payload(phantomAgent map[string]any) apitypes.TypedData {
	chainID := ethmath.NewHexOrDecimal256(1337)
	return apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			ChainId:           chainID,
			Name:              "Exchange",
			VerifyingContract: "0x0000000000000000000000000000000000000000",
			Version:           "1",
		},
		Types: apitypes.Types{
			"Agent": []apitypes.Type{
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
		},
		PrimaryType: "Agent",
		Message: apitypes.TypedDataMessage{
			"source":       phantomAgent["source"],
			"connectionId": hexutil.Bytes(phantomAgent["connectionId"].([]byte)),
		},
	}
}

// UserSignedPayload corresponds to Python:
// hyperliquid.utils.signing.user_signed_payload
func UserSignedPayload(primaryType string, payloadTypes []SignType, action OrderedMap) (apitypes.TypedData, error) {
	signatureChainID, ok := action.Get("signatureChainId")
	if !ok {
		return apitypes.TypedData{}, fmt.Errorf("signatureChainId missing")
	}
	chainIDText, ok := signatureChainID.(string)
	if !ok {
		return apitypes.TypedData{}, fmt.Errorf("signatureChainId must be string")
	}
	chainID, ok := new(big.Int).SetString(strings.TrimPrefix(chainIDText, "0x"), 16)
	if !ok {
		return apitypes.TypedData{}, fmt.Errorf("invalid signatureChainId: %s", chainIDText)
	}
	types := apitypes.Types{
		primaryType:    make([]apitypes.Type, 0, len(payloadTypes)),
		"EIP712Domain": eip712DomainTypes(),
	}
	for _, payloadType := range payloadTypes {
		types[primaryType] = append(types[primaryType], apitypes.Type{Name: payloadType.Name, Type: payloadType.Type})
	}
	return apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			Name:              "HyperliquidSignTransaction",
			Version:           "1",
			ChainId:           (*ethmath.HexOrDecimal256)(chainID),
			VerifyingContract: "0x0000000000000000000000000000000000000000",
		},
		Types:       types,
		PrimaryType: primaryType,
		Message:     action.TypedDataMessageForTypes(payloadTypes),
	}, nil
}

func eip712DomainTypes() []apitypes.Type {
	return []apitypes.Type{
		{Name: "name", Type: "string"},
		{Name: "version", Type: "string"},
		{Name: "chainId", Type: "uint256"},
		{Name: "verifyingContract", Type: "address"},
	}
}

// SignL1Action corresponds to Python:
// hyperliquid.utils.signing.sign_l1_action
func SignL1Action(wallet *ecdsa.PrivateKey, action any, activePool *string, nonce int64, expiresAfter *int64, isMainnet bool) (Signature, error) {
	hash, err := ActionHash(action, activePool, nonce, expiresAfter)
	if err != nil {
		return Signature{}, err
	}
	phantomAgent := ConstructPhantomAgent(hash, isMainnet)
	return SignInner(wallet, L1Payload(phantomAgent))
}

// SignUserSignedAction corresponds to Python:
// hyperliquid.utils.signing.sign_user_signed_action
func SignUserSignedAction(wallet *ecdsa.PrivateKey, action OrderedMap, payloadTypes []SignType, primaryType string, isMainnet bool) (Signature, OrderedMap, error) {
	action = action.Set("signatureChainId", "0x66eee")
	chain := "Testnet"
	if isMainnet {
		chain = "Mainnet"
	}
	action = action.Set("hyperliquidChain", chain)
	data, err := UserSignedPayload(primaryType, payloadTypes, action)
	if err != nil {
		return Signature{}, nil, err
	}
	sig, err := SignInner(wallet, data)
	if err != nil {
		return Signature{}, nil, err
	}
	return sig, action, nil
}

// SignUSDTransferAction corresponds to Python:
// hyperliquid.utils.signing.sign_usd_transfer_action
func SignUSDTransferAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, USDSendSignTypes, "HyperliquidTransaction:UsdSend", isMainnet)
}

// SignWithdrawFromBridgeAction corresponds to Python:
// hyperliquid.utils.signing.sign_withdraw_from_bridge_action
func SignWithdrawFromBridgeAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, WithdrawSignTypes, "HyperliquidTransaction:Withdraw", isMainnet)
}

// SignSpotTransferAction corresponds to Python:
// hyperliquid.utils.signing.sign_spot_transfer_action
func SignSpotTransferAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, SpotTransferSignTypes, "HyperliquidTransaction:SpotSend", isMainnet)
}

// SignUSDClassTransferAction corresponds to Python:
// hyperliquid.utils.signing.sign_usd_class_transfer_action
func SignUSDClassTransferAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, USDClassTransferSignTypes, "HyperliquidTransaction:UsdClassTransfer", isMainnet)
}

// SignSendAssetAction corresponds to Python:
// hyperliquid.utils.signing.sign_send_asset_action
func SignSendAssetAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, SendAssetSignTypes, "HyperliquidTransaction:SendAsset", isMainnet)
}

// SignUserDexAbstractionAction corresponds to Python:
// hyperliquid.utils.signing.sign_user_dex_abstraction_action
func SignUserDexAbstractionAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, UserDexAbstractionSignTypes, "HyperliquidTransaction:UserDexAbstraction", isMainnet)
}

// SignUserSetAbstractionAction corresponds to Python:
// hyperliquid.utils.signing.sign_user_set_abstraction_action
func SignUserSetAbstractionAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, UserSetAbstractionSignTypes, "HyperliquidTransaction:UserSetAbstraction", isMainnet)
}

// SignConvertToMultiSigUserAction corresponds to Python:
// hyperliquid.utils.signing.sign_convert_to_multi_sig_user_action
func SignConvertToMultiSigUserAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, ConvertToMultiSigUserSignTypes, "HyperliquidTransaction:ConvertToMultiSigUser", isMainnet)
}

// SignMultiSigAction corresponds to Python:
// hyperliquid.utils.signing.sign_multi_sig_action
func SignMultiSigAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool, vaultAddress *string, nonce int64, expiresAfter *int64) (Signature, error) {
	actionWithoutTag := action.Remove("type")
	multiSigActionHash, err := ActionHash(actionWithoutTag, vaultAddress, nonce, expiresAfter)
	if err != nil {
		return Signature{}, err
	}
	envelope := OrderedMap{
		{Key: "multiSigActionHash", Value: multiSigActionHash},
		{Key: "nonce", Value: uint64(nonce)},
	}
	signature, _, err := SignUserSignedAction(wallet, envelope, MultiSigEnvelopeSignTypes, "HyperliquidTransaction:SendMultiSig", isMainnet)
	return signature, err
}

// SignAgent corresponds to Python:
// hyperliquid.utils.signing.sign_agent
func SignAgent(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "agentAddress", Type: "address"},
		{Name: "agentName", Type: "string"},
		{Name: "nonce", Type: "uint64"},
	}, "HyperliquidTransaction:ApproveAgent", isMainnet)
}

// SignApproveBuilderFee corresponds to Python:
// hyperliquid.utils.signing.sign_approve_builder_fee
func SignApproveBuilderFee(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, []SignType{
		{Name: "hyperliquidChain", Type: "string"},
		{Name: "maxFeeRate", Type: "string"},
		{Name: "builder", Type: "address"},
		{Name: "nonce", Type: "uint64"},
	}, "HyperliquidTransaction:ApproveBuilderFee", isMainnet)
}

// SignTokenDelegateAction corresponds to Python:
// hyperliquid.utils.signing.sign_token_delegate_action
func SignTokenDelegateAction(wallet *ecdsa.PrivateKey, action OrderedMap, isMainnet bool) (Signature, OrderedMap, error) {
	return SignUserSignedAction(wallet, action, TokenDelegateSignTypes, "HyperliquidTransaction:TokenDelegate", isMainnet)
}

// SignInner corresponds to Python:
// hyperliquid.utils.signing.sign_inner
func SignInner(wallet *ecdsa.PrivateKey, data apitypes.TypedData) (Signature, error) {
	hash, _, err := apitypes.TypedDataAndHash(data)
	if err != nil {
		return Signature{}, err
	}
	sig, err := crypto.Sign(hash, wallet)
	if err != nil {
		return Signature{}, err
	}
	return Signature{
		R: intBytesToPythonHex(sig[:32]),
		S: intBytesToPythonHex(sig[32:64]),
		V: int(sig[64]) + 27,
	}, nil
}

func intBytesToPythonHex(b []byte) string {
	return "0x" + new(big.Int).SetBytes(b).Text(16)
}

// MarshalMsgpack preserves Python dict insertion order.
func (m OrderedMap) MarshalMsgpack() ([]byte, error) {
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	if err := enc.EncodeMapLen(len(m)); err != nil {
		return nil, err
	}
	for _, field := range m {
		if err := enc.EncodeString(field.Key); err != nil {
			return nil, err
		}
		if err := encodeMsgpackValue(enc, field.Value); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func encodeMsgpackValue(enc *msgpack.Encoder, value any) error {
	switch v := value.(type) {
	case OrderedMap:
		data, err := v.MarshalMsgpack()
		if err != nil {
			return err
		}
		_, err = enc.Writer().Write(data)
		return err
	case OrderWire:
		data, err := v.MarshalMsgpack()
		if err != nil {
			return err
		}
		_, err = enc.Writer().Write(data)
		return err
	case *big.Int:
		if v.Sign() >= 0 && v.BitLen() <= 64 {
			return enc.EncodeUint64(v.Uint64())
		}
		if v.IsInt64() {
			return enc.EncodeInt64(v.Int64())
		}
		return fmt.Errorf("big.Int out of msgpack integer range: %s", v.String())
	case []OrderWire:
		if err := enc.EncodeArrayLen(len(v)); err != nil {
			return err
		}
		for _, item := range v {
			if err := encodeMsgpackValue(enc, item); err != nil {
				return err
			}
		}
		return nil
	case []OrderedMap:
		if err := enc.EncodeArrayLen(len(v)); err != nil {
			return err
		}
		for _, item := range v {
			if err := encodeMsgpackValue(enc, item); err != nil {
				return err
			}
		}
		return nil
	default:
		return enc.Encode(value)
	}
}

// MarshalJSON preserves Python dict insertion order when action payloads are posted.
func (m OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, field := range m {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(field.Key)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(field.Value)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// Set returns a copy with key set, appending the key if it does not exist.
func (m OrderedMap) Set(key string, value any) OrderedMap {
	out := append(OrderedMap{}, m...)
	for i, field := range out {
		if field.Key == key {
			out[i].Value = value
			return out
		}
	}
	return append(out, Field{Key: key, Value: value})
}

// Remove returns a copy without key.
func (m OrderedMap) Remove(key string) OrderedMap {
	out := make(OrderedMap, 0, len(m))
	for _, field := range m {
		if field.Key != key {
			out = append(out, field)
		}
	}
	return out
}

// TypedDataMessage converts an ordered payload to a go-ethereum typed data message.
func (m OrderedMap) TypedDataMessage() apitypes.TypedDataMessage {
	msg := apitypes.TypedDataMessage{}
	for _, field := range m {
		msg[field.Key] = field.Value
	}
	return msg
}

// TypedDataMessageForTypes converts only declared typed fields.
func (m OrderedMap) TypedDataMessageForTypes(signTypes []SignType) apitypes.TypedDataMessage {
	msg := apitypes.TypedDataMessage{}
	for _, signType := range signTypes {
		if value, ok := m.Get(signType.Name); ok {
			if strings.HasPrefix(signType.Type, "uint") || strings.HasPrefix(signType.Type, "int") {
				msg[signType.Name] = typedIntegerValue(value)
			} else if strings.HasPrefix(signType.Type, "bytes") {
				msg[signType.Name] = typedBytesValue(value)
			} else {
				msg[signType.Name] = value
			}
		}
	}
	return msg
}

func typedBytesValue(value any) any {
	switch v := value.(type) {
	case []byte:
		return hexutil.Bytes(v)
	default:
		return value
	}
}

func typedIntegerValue(value any) any {
	switch v := value.(type) {
	case uint64:
		return new(big.Int).SetUint64(v)
	case uint:
		return new(big.Int).SetUint64(uint64(v))
	case int64:
		return big.NewInt(v)
	case int:
		return big.NewInt(int64(v))
	default:
		return value
	}
}

// MarshalMsgpack preserves Python dict insertion order for OrderWire.
func (w OrderWire) MarshalMsgpack() ([]byte, error) {
	return OrderedMap(w).MarshalMsgpack()
}

// MarshalJSON preserves Python dict insertion order for OrderWire.
func (w OrderWire) MarshalJSON() ([]byte, error) {
	return OrderedMap(w).MarshalJSON()
}

// Get returns the value for key. It is a test and porting convenience.
func (m OrderedMap) Get(key string) (any, bool) {
	for _, field := range m {
		if field.Key == key {
			return field.Value, true
		}
	}
	return nil, false
}

// OrderTypeToWire corresponds to Python:
// hyperliquid.utils.signing.order_type_to_wire
func OrderTypeToWire(orderType OrderType) (OrderedMap, error) {
	if limit, ok := orderType["limit"]; ok {
		return OrderedMap{{Key: "limit", Value: limit}}, nil
	}
	if triggerRaw, ok := orderType["trigger"]; ok {
		trigger, ok := triggerRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("trigger order type must be map[string]any")
		}
		triggerPx, ok := trigger["triggerPx"].(float64)
		if !ok {
			return nil, fmt.Errorf("triggerPx must be float64")
		}
		wirePx, err := FloatToWire(triggerPx)
		if err != nil {
			return nil, err
		}
		return OrderedMap{{
			Key: "trigger",
			Value: OrderedMap{
				{Key: "isMarket", Value: trigger["isMarket"]},
				{Key: "triggerPx", Value: wirePx},
				{Key: "tpsl", Value: trigger["tpsl"]},
			},
		}}, nil
	}
	return nil, fmt.Errorf("invalid order type: %v", orderType)
}

// OrderRequestToOrderWire corresponds to Python:
// hyperliquid.utils.signing.order_request_to_order_wire
func OrderRequestToOrderWire(order OrderRequest, asset int) (OrderWire, error) {
	px, err := FloatToWire(order.LimitPx)
	if err != nil {
		return nil, err
	}
	sz, err := FloatToWire(order.Sz)
	if err != nil {
		return nil, err
	}
	t, err := OrderTypeToWire(order.OrderType)
	if err != nil {
		return nil, err
	}
	wire := OrderWire{
		{Key: "a", Value: asset},
		{Key: "b", Value: order.IsBuy},
		{Key: "p", Value: px},
		{Key: "s", Value: sz},
		{Key: "r", Value: order.ReduceOnly},
		{Key: "t", Value: t},
	}
	if order.Cloid != nil {
		wire = append(wire, Field{Key: "c", Value: order.Cloid.ToRaw()})
	}
	return wire, nil
}

// OrderWiresToOrderAction corresponds to Python:
// hyperliquid.utils.signing.order_wires_to_order_action
func OrderWiresToOrderAction(orderWires []OrderWire, builder *types.BuilderInfo, grouping any) OrderedMap {
	if grouping == nil {
		grouping = "na"
	}
	action := OrderedMap{
		{Key: "type", Value: "order"},
		{Key: "orders", Value: orderWires},
		{Key: "grouping", Value: grouping},
	}
	if builder != nil {
		action = append(action, Field{Key: "builder", Value: OrderedMap{
			{Key: "b", Value: builder.B},
			{Key: "f", Value: builder.F},
		}})
	}
	return action
}
