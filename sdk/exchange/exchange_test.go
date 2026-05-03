package exchange

import (
	"context"
	"testing"

	"hyperliquid-bot/sdk/info"
)

func TestSlippagePriceWithProvidedPxMatchesPythonRounding(t *testing.T) {
	e := &Exchange{Info: info.New("", 0)}
	e.Info.NameToCoin["ETH"] = "ETH"
	e.Info.CoinToAsset["ETH"] = 1
	e.Info.AssetToSzDecimals[1] = 4
	px := 1670.1
	got, err := e.SlippagePrice(context.Background(), "ETH", true, 0.05, &px)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1753.6 {
		t.Fatalf("SlippagePrice=%v want 1753.6", got)
	}
}
