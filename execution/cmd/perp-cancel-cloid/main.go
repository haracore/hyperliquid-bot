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
		privateKey   = flag.String("private-key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"), "private key; can also be set with HYPERLIQUID_PRIVATE_KEY")
		baseURL      = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet      = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin         = flag.String("coin", "", "perp coin")
		dex          = flag.String("dex", "", "perp dex name; empty string is the default perp dex")
		cloidRaw     = flag.String("cloid", "", "client order id, 16-byte hex string")
		vaultAddress = flag.String("vault-address", os.Getenv("HYPERLIQUID_VAULT_ADDRESS"), "optional vault/subaccount address")
		confirm      = flag.Bool("confirm", false, "actually cancel the order")
		timeout      = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
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
	ex := clientutil.NewExchange(ctx, *privateKey, base, *dex, clientutil.OptionalString(vaultAddress), *timeout)
	var response any
	if err := ex.CancelByCloid(ctx, *coin, cloid, &response); err != nil {
		clientutil.ExitErr("perp cancel by cloid", err)
	}
	clientutil.PrintJSON(response)
}
