package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"hyperliquid-bot/execution/internal/clientutil"
)

func main() {
	var (
		privateKey = flag.String("private-key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"), "private key; can also be set with HYPERLIQUID_PRIVATE_KEY")
		baseURL    = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet    = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin       = flag.String("coin", "", "spot pair, for example PURR/USDC, or indexed spot asset like @8")
		cloidRaw   = flag.String("cloid", "", "client order id, 16-byte hex string")
		confirm    = flag.Bool("confirm", false, "actually cancel the order")
		timeout    = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	)
	flag.Parse()
	clientutil.RequirePrivateKey(*privateKey)
	clientutil.RequireCoin(*coin)
	if *cloidRaw == "" {
		clientutil.ExitUsage("missing -cloid")
	}
	if !*confirm {
		fmt.Fprintln(os.Stderr, "refusing to cancel without -confirm")
		os.Exit(2)
	}
	cloid := clientutil.ParseCloid(*cloidRaw, "cloid")
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	ex := clientutil.NewExchange(ctx, *privateKey, base, "", nil, *timeout)
	var response any
	if err := ex.CancelByCloid(ctx, *coin, cloid, &response); err != nil {
		clientutil.ExitErr("spot cancel by cloid", err)
	}
	clientutil.PrintJSON(response)
}
