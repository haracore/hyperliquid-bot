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
		address = flag.String("address", "", "user address; overrides execution secrets")
		baseURL = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet = flag.Bool("testnet", false, "use Hyperliquid testnet")
		dex     = flag.String("dex", "", "perp dex name; empty string is the default perp dex")
		showAll = flag.Bool("all", false, "show every returned position, including zero-size positions")
		timeout = flag.Duration("timeout", 15*time.Second, "HTTP timeout")
	)
	secretFlags := clientutil.AddSecretFlags()
	flag.Parse()

	account := clientutil.ResolveAccountFields(context.Background(), secretFlags, "", *address, "", *timeout)
	clientutil.RequireAddress(account.Address)
	base := clientutil.ResolveBaseURL(*baseURL, *testnet)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client := execution.New(execution.Config{BaseURL: base, Timeout: *timeout, Dex: *dex})
	positions, err := client.PerpPositions(ctx, account.Address, *showAll)
	if err != nil {
		clientutil.ExitErr("perp positions", err)
	}

	printHeader("Hyperliquid perp positions")
	fmt.Printf("Address: %s\n", account.Address)
	fmt.Printf("API:     %s\n", base)
	fmt.Printf("Dex:     %q\n", *dex)

	if len(positions) == 0 {
		fmt.Println()
		fmt.Println("(none)")
		return
	}

	fmt.Println()
	fmt.Printf("%-14s %14s %14s %14s %14s %12s\n", "coin", "szi", "entryPx", "positionValue", "unrealizedPnl", "leverage")
	fmt.Println(strings.Repeat("-", 88))
	for _, p := range positions {
		fmt.Printf(
			"%-14s %14s %14s %14s %14s %12s\n",
			p.Coin,
			p.Szi,
			p.EntryPx,
			p.PositionValue,
			p.UnrealizedPnl,
			p.Leverage,
		)
	}
}

func printHeader(title string) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
}
