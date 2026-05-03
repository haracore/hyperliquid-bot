package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"hyperliquid-bot/sdk/constants"
	"hyperliquid-bot/sdk/exchange"
	hlinfo "hyperliquid-bot/sdk/info"
	"hyperliquid-bot/sdk/signing"
	"hyperliquid-bot/sdk/types"
)

func main() {
	var (
		privateKey   = flag.String("private-key", os.Getenv("HYPERLIQUID_PRIVATE_KEY"), "private key; can also be set with HYPERLIQUID_PRIVATE_KEY")
		baseURL      = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet      = flag.Bool("testnet", false, "use Hyperliquid testnet")
		coin         = flag.String("coin", "", "perp coin, for example BTC or ETH")
		dex          = flag.String("dex", "", "perp dex name for metadata; empty string is the default perp dex")
		side         = flag.String("side", "buy", "buy or sell")
		size         = flag.Float64("size", 0, "order size")
		price        = flag.Float64("price", 0, "limit price")
		tif          = flag.String("tif", "Gtc", "time in force: Gtc, Ioc, or Alo")
		reduceOnly   = flag.Bool("reduce-only", false, "place reduce-only order")
		vaultAddress = flag.String("vault-address", os.Getenv("HYPERLIQUID_VAULT_ADDRESS"), "optional vault/subaccount address")
		cloidRaw     = flag.String("cloid", "", "optional client order id, 16-byte hex string")
		confirm      = flag.Bool("confirm", false, "actually submit the order")
		timeout      = flag.Duration("timeout", 20*time.Second, "HTTP timeout")
	)
	flag.Parse()

	if strings.TrimSpace(*privateKey) == "" {
		exitUsage("missing private key: pass -private-key or set HYPERLIQUID_PRIVATE_KEY")
	}
	if strings.TrimSpace(*coin) == "" {
		exitUsage("missing -coin")
	}
	if *size <= 0 {
		exitUsage("-size must be greater than 0")
	}
	if *price <= 0 {
		exitUsage("-price must be greater than 0")
	}
	isBuy, err := parseSide(*side)
	if err != nil {
		exitUsage(err.Error())
	}
	if !validTIF(*tif) {
		exitUsage("-tif must be one of Gtc, Ioc, Alo")
	}
	if *baseURL == "" {
		*baseURL = constants.MainnetAPIURL
	}
	if *testnet {
		*baseURL = constants.TestnetAPIURL
	}
	if !*confirm {
		fmt.Fprintln(os.Stderr, "refusing to submit without -confirm")
		fmt.Fprintln(os.Stderr, "review the command, then add -confirm to place the order")
		os.Exit(2)
	}

	wallet, err := signing.PrivateKeyFromHex(*privateKey)
	if err != nil {
		exitErr("private key", err)
	}

	var cloid *types.Cloid
	if strings.TrimSpace(*cloidRaw) != "" {
		parsed, err := types.NewCloid(*cloidRaw)
		if err != nil {
			exitErr("cloid", err)
		}
		cloid = &parsed
	}

	var vault *string
	if strings.TrimSpace(*vaultAddress) != "" {
		vault = vaultAddress
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	perpDexs := []string{""}
	if strings.TrimSpace(*dex) != "" {
		perpDexs = []string{*dex}
	}
	info, err := hlinfo.NewInitialized(ctx, *baseURL, true, nil, nil, perpDexs, *timeout)
	if err != nil {
		exitErr("initialize info metadata", err)
	}

	ex := exchange.New(wallet, *baseURL, *timeout, vault, nil)
	ex.Info = info

	var response any
	err = ex.Order(
		ctx,
		*coin,
		isBuy,
		*size,
		*price,
		signing.OrderType{"limit": map[string]any{"tif": *tif}},
		*reduceOnly,
		cloid,
		nil,
		&response,
	)
	if err != nil {
		exitErr("place perp order", err)
	}

	pretty, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		exitErr("format response", err)
	}
	fmt.Println(string(pretty))
}

func parseSide(side string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "buy", "b":
		return true, nil
	case "sell", "s":
		return false, nil
	default:
		return false, fmt.Errorf("-side must be buy or sell")
	}
}

func validTIF(tif string) bool {
	switch tif {
	case "Gtc", "Ioc", "Alo":
		return true
	default:
		return false
	}
}

func exitUsage(message string) {
	fmt.Fprintln(os.Stderr, message)
	flag.Usage()
	os.Exit(2)
}

func exitErr(label string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", label, err)
	os.Exit(1)
}
