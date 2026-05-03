package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	execution "hyperliquid-bot/execution/client"
	"hyperliquid-bot/execution/internal/clientutil"
)

func main() {
	var (
		privateKey   = flag.String("private-key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"), "private key; can also be set with HYPERLIQUID_PRIVATE_KEY")
		baseURL      = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet      = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin         = flag.String("coin", "", "perp coin")
		dex          = flag.String("dex", "", "perp dex name; empty string is the default perp dex")
		oid          = flag.Int("oid", 0, "order id to cancel")
		vaultAddress = flag.String("vault-address", os.Getenv("HYPERLIQUID_VAULT_ADDRESS"), "optional vault/subaccount address")
		confirm      = flag.Bool("confirm", false, "actually cancel the order")
		timeout      = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	)
	flag.Parse()
	clientutil.RequirePrivateKey(*privateKey)
	clientutil.RequireCoin(*coin)
	if *oid == 0 {
		clientutil.ExitUsage("missing -oid")
	}
	if !*confirm {
		fmt.Fprintln(os.Stderr, "refusing to cancel without -confirm")
		os.Exit(2)
	}
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := execution.New(execution.Config{
		BaseURL:      base,
		Timeout:      *timeout,
		PrivateKey:   *privateKey,
		Dex:          *dex,
		VaultAddress: clientutil.OptionalString(vaultAddress),
	})
	response, err := client.CancelPerpOrder(ctx, execution.CancelOrderRequest{Coin: *coin, Oid: *oid})
	if err != nil {
		clientutil.ExitErr("perp cancel order", err)
	}
	clientutil.PrintJSON(response)
}
