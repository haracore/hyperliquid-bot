package signing

import (
	"encoding/hex"
	"testing"

	"hyperliquid-bot/sdk/types"
)

func TestFloatToIntForHashingMatchesPython(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{123123123123, "12312312312300000000"},
		{0.00001231, "1231"},
		{1.033, "103300000"},
	}
	for _, tt := range tests {
		got, err := FloatToIntForHashing(tt.in)
		if err != nil {
			t.Fatalf("FloatToIntForHashing(%v): %v", tt.in, err)
		}
		if got.String() != tt.want {
			t.Fatalf("FloatToIntForHashing(%v)=%s want %s", tt.in, got.String(), tt.want)
		}
	}
	if _, err := FloatToIntForHashing(0.000012312312); err == nil {
		t.Fatal("expected rounding error")
	}
}

func TestPhantomAgentCreationMatchesPython(t *testing.T) {
	order := OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         0.0147,
		LimitPx:    1670.1,
		ReduceOnly: false,
		OrderType:  OrderType{"limit": map[string]any{"tif": "Ioc"}},
	}
	wire, err := OrderRequestToOrderWire(order, 4)
	if err != nil {
		t.Fatal(err)
	}
	action := OrderWiresToOrderAction([]OrderWire{wire}, nil, nil)
	hash, err := ActionHash(action, nil, 1677777606040, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := "0x" + hex.EncodeToString(ConstructPhantomAgent(hash, true)["connectionId"].([]byte))
	want := "0x0fcbeda5ae3c4950a548021552a4fea2226858c4453571bf3f24ba017eac2908"
	if got != want {
		t.Fatalf("connectionId=%s want %s", got, want)
	}
}

func TestL1ActionSigningMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	num, err := FloatToIntForHashing(1000)
	if err != nil {
		t.Fatal(err)
	}
	action := OrderedMap{
		{Key: "type", Value: "dummy"},
		{Key: "num", Value: num},
	}
	mainnet, err := SignL1Action(wallet, action, nil, 0, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.R != "0x53749d5b30552aeb2fca34b530185976545bb22d0b3ce6f62e31be961a59298" ||
		mainnet.S != "0x755c40ba9bf05223521753995abb2f73ab3229be8ec921f350cb447e384d8ed8" ||
		mainnet.V != 27 {
		t.Fatalf("unexpected mainnet signature: %+v", mainnet)
	}
	testnet, err := SignL1Action(wallet, action, nil, 0, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if testnet.R != "0x542af61ef1f429707e3c76c5293c80d01f74ef853e34b76efffcb57e574f9510" ||
		testnet.S != "0x17b8b32f086e8cdede991f1e2c529f5dd5297cbe8128500e00cbaf766204a613" ||
		testnet.V != 28 {
		t.Fatalf("unexpected testnet signature: %+v", testnet)
	}
}

func TestL1ActionSigningOrderMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := OrderRequestToOrderWire(OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         100,
		LimitPx:    100,
		ReduceOnly: false,
		OrderType:  OrderType{"limit": map[string]any{"tif": "Gtc"}},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	action := OrderWiresToOrderAction([]OrderWire{wire}, nil, nil)

	mainnet, err := SignL1Action(wallet, action, nil, 0, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.R != "0xd65369825a9df5d80099e513cce430311d7d26ddf477f5b3a33d2806b100d78e" ||
		mainnet.S != "0x2b54116ff64054968aa237c20ca9ff68000f977c93289157748a3162b6ea940e" ||
		mainnet.V != 28 {
		t.Fatalf("unexpected mainnet signature: %+v", mainnet)
	}

	testnet, err := SignL1Action(wallet, action, nil, 0, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if testnet.R != "0x82b2ba28e76b3d761093aaded1b1cdad4960b3af30212b343fb2e6cdfa4e3d54" ||
		testnet.S != "0x6b53878fc99d26047f4d7e8c90eb98955a109f44209163f52d8dc4278cbbd9f5" ||
		testnet.V != 27 {
		t.Fatalf("unexpected testnet signature: %+v", testnet)
	}
}

func TestL1ActionSigningOrderWithCloidMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	cloid := types.CloidFromInt(1)
	wire, err := OrderRequestToOrderWire(OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         100,
		LimitPx:    100,
		ReduceOnly: false,
		OrderType:  OrderType{"limit": map[string]any{"tif": "Gtc"}},
		Cloid:      &cloid,
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	action := OrderWiresToOrderAction([]OrderWire{wire}, nil, nil)
	mainnet, err := SignL1Action(wallet, action, nil, 0, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.R != "0x41ae18e8239a56cacbc5dad94d45d0b747e5da11ad564077fcac71277a946e3" ||
		mainnet.S != "0x3c61f667e747404fe7eea8f90ab0e76cc12ce60270438b2058324681a00116da" ||
		mainnet.V != 27 {
		t.Fatalf("unexpected mainnet signature: %+v", mainnet)
	}
}

func TestL1ActionSigningMatchesWithVaultPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	num, err := FloatToIntForHashing(1000)
	if err != nil {
		t.Fatal(err)
	}
	action := OrderedMap{{Key: "type", Value: "dummy"}, {Key: "num", Value: num}}
	vault := "0x1719884eb866cb12b2287399b15f7db5e7d775ea"
	mainnet, err := SignL1Action(wallet, action, &vault, 0, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.R != "0x3c548db75e479f8012acf3000ca3a6b05606bc2ec0c29c50c515066a326239" ||
		mainnet.S != "0x4d402be7396ce74fbba3795769cda45aec00dc3125a984f2a9f23177b190da2c" ||
		mainnet.V != 28 {
		t.Fatalf("unexpected mainnet signature: %+v", mainnet)
	}
}

func TestL1ActionSigningTPSLOrderMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	wire, err := OrderRequestToOrderWire(OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         100,
		LimitPx:    100,
		ReduceOnly: false,
		OrderType: OrderType{"trigger": map[string]any{
			"triggerPx": float64(103),
			"isMarket":  true,
			"tpsl":      "sl",
		}},
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	action := OrderWiresToOrderAction([]OrderWire{wire}, nil, nil)
	mainnet, err := SignL1Action(wallet, action, nil, 0, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.R != "0x98343f2b5ae8e26bb2587daad3863bc70d8792b09af1841b6fdd530a2065a3f9" ||
		mainnet.S != "0x6b5bb6bb0633b710aa22b721dd9dee6d083646a5f8e581a20b545be6c1feb405" ||
		mainnet.V != 27 {
		t.Fatalf("unexpected mainnet signature: %+v", mainnet)
	}
}

func TestSignUSDTransferActionMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	action := OrderedMap{
		{Key: "destination", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
		{Key: "amount", Value: "1"},
		{Key: "time", Value: uint64(1687816341423)},
	}
	signature, _, err := SignUSDTransferAction(wallet, action, false)
	if err != nil {
		t.Fatal(err)
	}
	if signature.R != "0x637b37dd731507cdd24f46532ca8ba6eec616952c56218baeff04144e4a77073" ||
		signature.S != "0x11a6a24900e6e314136d2592e2f8d502cd89b7c15b198e1bee043c9589f9fad7" ||
		signature.V != 27 {
		t.Fatalf("unexpected signature: %+v", signature)
	}
}

func TestSignWithdrawFromBridgeActionMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	action := OrderedMap{
		{Key: "destination", Value: "0x5e9ee1089755c3435139848e47e6635505d5a13a"},
		{Key: "amount", Value: "1"},
		{Key: "time", Value: uint64(1687816341423)},
	}
	signature, _, err := SignWithdrawFromBridgeAction(wallet, action, false)
	if err != nil {
		t.Fatal(err)
	}
	if signature.R != "0x8363524c799e90ce9bc41022f7c39b4e9bdba786e5f9c72b20e43e1462c37cf9" ||
		signature.S != "0x58b1411a775938b83e29182e8ef74975f9054c8e97ebf5ec2dc8d51bfc893881" ||
		signature.V != 28 {
		t.Fatalf("unexpected signature: %+v", signature)
	}
}

func TestCreateSubAccountActionMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	action := OrderedMap{{Key: "type", Value: "createSubAccount"}, {Key: "name", Value: "example"}}
	mainnet, err := SignL1Action(wallet, action, nil, 0, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.R != "0x51096fe3239421d16b671e192f574ae24ae14329099b6db28e479b86cdd6caa7" ||
		mainnet.S != "0xb71f7d293af92d3772572afb8b102d167a7cef7473388286bc01f52a5c5b423" ||
		mainnet.V != 27 {
		t.Fatalf("unexpected mainnet signature: %+v", mainnet)
	}
}

func TestScheduleCancelActionMatchesPython(t *testing.T) {
	wallet, err := PrivateKeyFromHex("0x0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	action := OrderedMap{{Key: "type", Value: "scheduleCancel"}}
	mainnet, err := SignL1Action(wallet, action, nil, 0, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if mainnet.R != "0x6cdfb286702f5917e76cd9b3b8bf678fcc49aec194c02a73e6d4f16891195df9" ||
		mainnet.S != "0x6557ac307fa05d25b8d61f21fb8a938e703b3d9bf575f6717ba21ec61261b2a0" ||
		mainnet.V != 27 {
		t.Fatalf("unexpected mainnet signature: %+v", mainnet)
	}
}

func TestOrderWireWithCloidMatchesPythonShape(t *testing.T) {
	cloid := types.CloidFromInt(1)
	wire, err := OrderRequestToOrderWire(OrderRequest{
		Coin:       "ETH",
		IsBuy:      true,
		Sz:         100,
		LimitPx:    100,
		ReduceOnly: false,
		OrderType:  OrderType{"limit": map[string]any{"tif": "Gtc"}},
		Cloid:      &cloid,
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	cloidValue, _ := OrderedMap(wire).Get("c")
	if cloidValue != "0x00000000000000000000000000000001" {
		t.Fatalf("unexpected cloid wire: %v", cloidValue)
	}
	pValue, _ := OrderedMap(wire).Get("p")
	sValue, _ := OrderedMap(wire).Get("s")
	if pValue != "100" || sValue != "100" {
		t.Fatalf("unexpected price/size wire: %v", wire)
	}
}
