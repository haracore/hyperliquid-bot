package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	execution "hyperliquid-bot/execution/client"
	"hyperliquid-bot/execution/internal/clientutil"
)

func main() {
	var (
		privateKey   = flag.String("private-key", "", "private key; overrides execution secrets")
		baseURL      = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet      = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin         = flag.String("coin", "", "perp coin, for example BTC or ETH")
		dex          = flag.String("dex", "", "perp dex name for metadata; empty string is the default perp dex")
		side         = flag.String("side", "buy", "buy or sell")
		size         = flag.Float64("size", 0, "order size")
		price        = flag.Float64("price", 0, "limit price")
		tif          = flag.String("tif", "Gtc", "time in force: Gtc, Ioc, or Alo")
		reduceOnly   = flag.Bool("reduce-only", false, "place reduce-only order")
		vaultAddress = flag.String("vault-address", "", "optional vault/subaccount address; overrides execution secrets")
		cloidRaw     = flag.String("cloid", "", "optional client order id, 16-byte hex string")
		confirm      = flag.Bool("confirm", false, "actually submit the order")
		timeout      = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	)
	secretFlags := clientutil.AddSecretFlags()
	flag.Parse()

	account := clientutil.ResolveAccount(context.Background(), secretFlags, *privateKey, "", *vaultAddress, *timeout)
	clientutil.RequirePrivateKey(account.PrivateKey)
	clientutil.RequireCoin(*coin)
	if *size <= 0 {
		clientutil.ExitUsage("-size must be greater than 0")
	}
	if *price <= 0 {
		clientutil.ExitUsage("-price must be greater than 0")
	}
	isBuy, err := clientutil.ParseSide(*side)
	if err != nil {
		clientutil.ExitUsage(err.Error())
	}
	if !clientutil.ValidTIF(*tif) {
		clientutil.ExitUsage("-tif must be one of Gtc, Ioc, Alo")
	}
	if !*confirm {
		fmt.Fprintln(os.Stderr, "refusing to submit without -confirm")
		fmt.Fprintln(os.Stderr, "review the command, then add -confirm to place the order")
		os.Exit(2)
	}

	var cloid *execution.Cloid
	if strings.TrimSpace(*cloidRaw) != "" {
		parsed, err := execution.NewCloid(*cloidRaw)
		if err != nil {
			clientutil.ExitErr("cloid", err)
		}
		cloid = &parsed
	}

	base := clientutil.ResolveBaseURL(*baseURL, *testnet)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client := execution.New(execution.Config{
		BaseURL:      base,
		Timeout:      *timeout,
		PrivateKey:   account.PrivateKey,
		Dex:          *dex,
		VaultAddress: clientutil.OptionalString(&account.VaultAddress),
	})
	response, err := client.PlacePerpOrder(ctx, execution.OrderRequest{
		Coin:       *coin,
		IsBuy:      isBuy,
		Size:       *size,
		Price:      *price,
		TIF:        *tif,
		ReduceOnly: *reduceOnly,
		Cloid:      cloid,
	})
	if err != nil {
		clientutil.ExitErr("place perp order", err)
	}
	clientutil.PrintJSON(response)
}
