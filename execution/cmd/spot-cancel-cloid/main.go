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
		privateKey = flag.String("private-key", "", "private key; overrides execution secrets")
		baseURL    = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet    = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin       = flag.String("coin", "", "spot pair, for example PURR/USDC, or indexed spot asset like @8")
		cloidRaw   = flag.String("cloid", "", "client order id, 16-byte hex string")
		confirm    = flag.Bool("confirm", false, "actually cancel the order")
		timeout    = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	)
	secretFlags := clientutil.AddSecretFlags()
	flag.Parse()
	account := clientutil.ResolveAccount(context.Background(), secretFlags, *privateKey, "", "", *timeout)
	clientutil.RequirePrivateKey(account.PrivateKey)
	clientutil.RequireCoin(*coin)
	if *cloidRaw == "" {
		clientutil.ExitUsage("missing -cloid")
	}
	if !*confirm {
		fmt.Fprintln(os.Stderr, "refusing to cancel without -confirm")
		os.Exit(2)
	}
	cloid, err := execution.NewCloid(*cloidRaw)
	if err != nil {
		clientutil.ExitErr("cloid", err)
	}
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client := execution.New(execution.Config{BaseURL: base, Timeout: *timeout, PrivateKey: account.PrivateKey})
	response, err := client.CancelSpotByCloid(ctx, execution.CancelByCloidRequest{Coin: *coin, Cloid: cloid})
	if err != nil {
		clientutil.ExitErr("spot cancel by cloid", err)
	}
	clientutil.PrintJSON(response)
}
