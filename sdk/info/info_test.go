package info

import (
	"testing"

	ws "hyperliquid-bot/sdk/websocket"
)

func TestRemapCoinSubscriptionMatchesPython(t *testing.T) {
	i := New("", 0)
	i.NameToCoin["PURR/USDC"] = "PURR"
	sub := ws.Subscription{"type": "l2Book", "coin": "PURR/USDC"}
	i.RemapCoinSubscription(sub)
	if sub["coin"] != "PURR" {
		t.Fatalf("coin=%v want PURR", sub["coin"])
	}
}
