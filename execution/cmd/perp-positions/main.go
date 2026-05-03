package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"hyperliquid-bot/sdk/constants"
	hlinfo "hyperliquid-bot/sdk/info"
)

func main() {
	var (
		address = flag.String("address", os.Getenv("HYPERLIQUID_ADDRESS"), "user address; can also be set with HYPERLIQUID_ADDRESS")
		baseURL = flag.String("base-url", os.Getenv("HYPERLIQUID_BASE_URL"), "Hyperliquid API base URL")
		testnet = flag.Bool("testnet", false, "use Hyperliquid testnet")
		dex     = flag.String("dex", "", "perp dex name; empty string is the default perp dex")
		showAll = flag.Bool("all", false, "show every returned position, including zero-size positions")
		timeout = flag.Duration("timeout", 15*time.Second, "HTTP timeout")
	)
	flag.Parse()

	if strings.TrimSpace(*address) == "" {
		fmt.Fprintln(os.Stderr, "missing address: pass -address or set HYPERLIQUID_ADDRESS")
		os.Exit(2)
	}
	if *baseURL == "" {
		*baseURL = constants.MainnetAPIURL
	}
	if *testnet {
		*baseURL = constants.TestnetAPIURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	info := hlinfo.New(*baseURL, *timeout)
	var state map[string]any
	if err := info.UserState(ctx, *address, *dex, &state); err != nil {
		fmt.Fprintf(os.Stderr, "user state: %v\n", err)
		os.Exit(1)
	}

	printHeader("Hyperliquid perp positions")
	fmt.Printf("Address: %s\n", *address)
	fmt.Printf("API:     %s\n", *baseURL)
	fmt.Printf("Dex:     %q\n", *dex)

	positions := extractPositions(state, *showAll)
	if len(positions) == 0 {
		fmt.Println()
		fmt.Println("(none)")
		return
	}

	sort.Slice(positions, func(i, j int) bool {
		return positions[i].coin < positions[j].coin
	})

	fmt.Println()
	fmt.Printf("%-14s %14s %14s %14s %14s %12s\n", "coin", "szi", "entryPx", "positionValue", "unrealizedPnl", "leverage")
	fmt.Println(strings.Repeat("-", 88))
	for _, p := range positions {
		fmt.Printf(
			"%-14s %14s %14s %14s %14s %12s\n",
			p.coin,
			p.szi,
			p.entryPx,
			p.positionValue,
			p.unrealizedPnl,
			p.leverage,
		)
	}
}

type positionRow struct {
	coin          string
	szi           string
	entryPx       string
	positionValue string
	unrealizedPnl string
	leverage      string
}

func extractPositions(state map[string]any, showAll bool) []positionRow {
	rawPositions, _ := state["assetPositions"].([]any)
	rows := make([]positionRow, 0, len(rawPositions))
	for _, raw := range rawPositions {
		wrapper, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		position, ok := wrapper["position"].(map[string]any)
		if !ok {
			continue
		}
		szi := stringValue(position["szi"])
		if !showAll && isZero(szi) {
			continue
		}
		rows = append(rows, positionRow{
			coin:          stringValue(position["coin"]),
			szi:           szi,
			entryPx:       stringValue(position["entryPx"]),
			positionValue: stringValue(position["positionValue"]),
			unrealizedPnl: stringValue(position["unrealizedPnl"]),
			leverage:      leverageValue(position["leverage"]),
		})
	}
	return rows
}

func printHeader(title string) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func isZero(value string) bool {
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed == 0
}

func leverageValue(value any) string {
	lev, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	typ := stringValue(lev["type"])
	raw := stringValue(lev["value"])
	if typ == "" {
		return raw
	}
	if raw == "" {
		return typ
	}
	return typ + ":" + raw
}
