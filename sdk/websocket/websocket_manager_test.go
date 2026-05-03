package websocket

import "testing"

func TestSubscriptionToIdentifierMatchesPython(t *testing.T) {
	tests := []struct {
		sub  Subscription
		want string
	}{
		{Subscription{"type": "allMids"}, "allMids"},
		{Subscription{"type": "l2Book", "coin": "ETH"}, "l2Book:eth"},
		{Subscription{"type": "candle", "coin": "BTC", "interval": "1m"}, "candle:btc,1m"},
		{Subscription{"type": "activeAssetData", "coin": "BTC", "user": "0xABC"}, "activeAssetData:btc,0xabc"},
	}
	for _, tt := range tests {
		if got := SubscriptionToIdentifier(tt.sub); got != tt.want {
			t.Fatalf("SubscriptionToIdentifier(%v)=%s want %s", tt.sub, got, tt.want)
		}
	}
}

func TestWsMsgToIdentifierMatchesPython(t *testing.T) {
	msg := map[string]any{"channel": "activeSpotAssetCtx", "data": map[string]any{"coin": "PURR"}}
	if got := WsMsgToIdentifier(msg); got != "activeAssetCtx:purr" {
		t.Fatalf("WsMsgToIdentifier=%s want activeAssetCtx:purr", got)
	}
}
